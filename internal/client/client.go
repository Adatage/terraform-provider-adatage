// Package client calls the Adatage Cloud API: ConnectRPC with JSON bodies over HTTP, signed in as a service
// account (see docs/cloud-platform/service-accounts.md in the platform repository).
package client

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultAPI = "https://api.cloud.adatage.com"

type Credentials struct {
	Type             string `json:"type"`
	ServiceAccountID string `json:"service_account_id"`
	KeyID            string `json:"key_id"`
	PrivateKey       string `json:"private_key"`
	ProjectID        string `json:"project_id"`
	TokenURI         string `json:"token_uri"`
}

func ParseCredentials(text string) (*Credentials, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") {
		raw, err := os.ReadFile(text)
		if err != nil {
			return nil, fmt.Errorf("credentials: %w", err)
		}
		text = string(raw)
	}
	var c Credentials
	if err := json.Unmarshal([]byte(text), &c); err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}
	if c.ServiceAccountID == "" || c.KeyID == "" || c.PrivateKey == "" {
		return nil, errors.New("credentials: service_account_id, key_id and private_key are required")
	}
	return &c, nil
}

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && (e.Code == "not_found" || e.Status == http.StatusNotFound)
}

type Client struct {
	API     string
	Project string
	HTTP    *http.Client
	Agent   string

	creds *Credentials
	mu    sync.Mutex
	token string
	until time.Time
}

func New(api, project string, creds *Credentials, accessToken string) (*Client, error) {
	if api == "" {
		api = DefaultAPI
	}
	if project == "" && creds != nil {
		project = creds.ProjectID
	}
	if project == "" {
		return nil, errors.New("a project is required")
	}
	if creds == nil && accessToken == "" {
		return nil, errors.New("credentials or an access token are required")
	}
	c := &Client{API: strings.TrimRight(api, "/"), Project: project, HTTP: &http.Client{Timeout: 2 * time.Minute}, creds: creds, token: accessToken}
	if accessToken != "" {
		c.until = time.Now().Add(100 * 365 * 24 * time.Hour)
	}
	return c, nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (c *Client) assertion() (string, error) {
	block, _ := pem.Decode([]byte(c.creds.PrivateKey))
	if block == nil {
		return "", errors.New("credentials: private_key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("credentials: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("credentials: private_key is not an RSA key")
	}
	aud := c.creds.TokenURI
	if aud == "" {
		aud = c.API + "/cloud.external.iam.v1.ServiceAccountTokenService/CreateServiceAccountToken"
	}
	jti := make([]byte, 16)
	rand.Read(jti)
	now := time.Now()
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": c.creds.KeyID})
	cl, _ := json.Marshal(map[string]any{"iss": c.creds.ServiceAccountID, "sub": c.creds.ServiceAccountID, "aud": aud, "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "jti": b64(jti)})
	signing := b64(h) + "." + b64(cl)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signing + "." + b64(sig), nil
}

func (c *Client) bearer(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.until) {
		return c.token, nil
	}
	if c.creds == nil {
		return "", errors.New("the access token has expired")
	}
	a, err := c.assertion()
	if err != nil {
		return "", err
	}
	aud := c.creds.TokenURI
	url := aud
	if url == "" {
		url = c.API + "/cloud.external.iam.v1.ServiceAccountTokenService/CreateServiceAccountToken"
	}
	var out struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   any    `json:"expiresIn"`
	}
	if err := c.post(ctx, url, "", nil, map[string]any{"assertion": a}, &out); err != nil {
		return "", fmt.Errorf("signing in: %w", err)
	}
	secs := 3600
	switch v := out.ExpiresIn.(type) {
	case float64:
		secs = int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			secs = n
		}
	}
	c.token = out.AccessToken
	c.until = time.Now().Add(time.Duration(secs)*time.Second - 5*time.Minute)
	return c.token, nil
}

func (c *Client) post(ctx context.Context, url, token string, extra map[string]string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var last error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		if c.Agent != "" {
			req.Header.Set("User-Agent", c.Agent)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		for k, v := range extra {
			req.Header.Set(k, v)
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			last = err
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(res.Body, 32<<20))
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			if out == nil || len(bytes.TrimSpace(data)) == 0 {
				return nil
			}
			return json.Unmarshal(data, out)
		}
		e := &Error{Status: res.StatusCode, Code: strings.ToLower(http.StatusText(res.StatusCode)), Message: strings.TrimSpace(string(data))}
		var ce struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &ce) == nil && ce.Code != "" {
			e.Code, e.Message = ce.Code, ce.Message
		}
		if res.StatusCode == http.StatusBadGateway || res.StatusCode == http.StatusServiceUnavailable || res.StatusCode == http.StatusGatewayTimeout || e.Code == "unavailable" {
			last = e
			continue
		}
		return e
	}
	return last
}

func (c *Client) Call(ctx context.Context, service, method string, body map[string]any) (map[string]any, error) {
	token, err := c.bearer(ctx)
	if err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]any{}
	}
	out := map[string]any{}
	err = c.post(ctx, c.API+"/"+service+"/"+method, token, map[string]string{"X-Project-Id": c.Project}, body, &out)
	return out, err
}

func (c *Client) WaitOperation(ctx context.Context, op map[string]any) (map[string]any, error) {
	name, _ := op["name"].(string)
	for {
		if done, _ := op["done"].(bool); done {
			if msg, _ := op["errorMessage"].(string); msg != "" {
				return op, fmt.Errorf("the operation failed: %s", msg)
			}
			return op, nil
		}
		if name == "" {
			return op, errors.New("the API returned no operation")
		}
		select {
		case <-ctx.Done():
			return op, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		var err error
		if op, err = c.Call(ctx, "cloud.external.compute.v1.OperationService", "GetOperation", map[string]any{"name": name}); err != nil {
			return op, err
		}
	}
}
