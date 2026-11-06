package client

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignInWithServiceAccountCredentials(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	privatePEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	tokens, calls := 0, 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/CreateServiceAccountToken") {
			var body struct{ Assertion string }
			json.NewDecoder(r.Body).Decode(&body)
			parts := strings.Split(body.Assertion, ".")
			sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
			sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
			if rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, sum[:], sig) != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"code":"unauthenticated","message":"bad signature"}`))
				return
			}
			var h map[string]string
			var c map[string]any
			hb, _ := base64.RawURLEncoding.DecodeString(parts[0])
			cb, _ := base64.RawURLEncoding.DecodeString(parts[1])
			json.Unmarshal(hb, &h)
			json.Unmarshal(cb, &c)
			if h["alg"] != "RS256" || h["kid"] != "22" || c["iss"] != "11" || c["sub"] != "11" || c["aud"] != srv.URL+"/token" || c["jti"] == "" {
				t.Errorf("header %v claims %v", h, c)
			}
			tokens++
			w.Write([]byte(`{"accessToken":"adtk_abc","expiresIn":"3600","tokenType":"Bearer"}`))
			return
		}
		calls++
		if r.Header.Get("Authorization") != "Bearer adtk_abc" || r.Header.Get("X-Project-Id") != "33" {
			t.Errorf("headers %v", r.Header)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	// The token endpoint path ends in the method name, as in the real credentials.
	creds := &Credentials{ServiceAccountID: "11", KeyID: "22", PrivateKey: privatePEM, ProjectID: "33", TokenURI: srv.URL + "/token"}
	creds.TokenURI = srv.URL + "/token"
	srv.Config.Handler = wrapPath(srv.Config.Handler)

	c, err := New(srv.URL, "", creds, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := c.Call(context.Background(), "svc.S", "M", nil); err != nil {
			t.Fatal(err)
		}
	}
	if tokens != 1 || calls != 3 {
		t.Fatalf("one sign-in for three calls: tokens %d calls %d", tokens, calls)
	}
}

// wrapPath lets the test's token_uri (/token) reach the token handler.
func wrapPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			r.URL.Path = "/x/CreateServiceAccountToken"
		}
		next.ServeHTTP(w, r)
	})
}

func TestParseCredentials(t *testing.T) {
	if _, err := ParseCredentials(`{"service_account_id":"1","key_id":"2","private_key":"x"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseCredentials(`{"service_account_id":"1"}`); err == nil {
		t.Fatal("a key is required")
	}
	if _, err := ParseCredentials("/no/such/file.json"); err == nil {
		t.Fatal("a missing file is an error")
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&Error{Code: "not_found"}) || IsNotFound(&Error{Code: "internal"}) || IsNotFound(nil) {
		t.Fatal("not found")
	}
}
