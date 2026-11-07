package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
)

const (
	svcCompute    = "cloud.external.compute.v1.ComputeService"
	svcDisk       = "cloud.external.compute.v1.DiskService"
	svcImage      = "cloud.external.compute.v1.ImageService"
	svcTemplate   = "cloud.external.compute.v1.TemplateService"
	svcSnapshot   = "cloud.external.compute.v1.SnapshotService"
	svcSshKey     = "cloud.external.compute.v1.SshKeyService"
	svcNetwork    = "cloud.external.network.v1.NetworkService"
	svcLB         = "cloud.external.network.v1.LoadBalancerService"
	svcKubernetes = "cloud.external.kubernetes.v1.KubernetesService"
	svcIam        = "cloud.external.iam.v1.IamService"
	svcAccounts   = "cloud.external.iam.v1.ServiceAccountService"
)

type states struct {
	path   string
	ready  []string
	failed []string
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func pollUntil(ctx context.Context, fn func() (bool, error)) error {
	for {
		done, err := fn()
		if err != nil || done {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("gave up waiting: %w", ctx.Err())
		case <-time.After(3 * time.Second):
		}
	}
}

func waitState(ctx context.Context, get func() (map[string]any, error), s states) error {
	return pollUntil(ctx, func() (bool, error) {
		obj, err := get()
		if err != nil {
			return false, err
		}
		v, _ := lookup(obj, s.path)
		state, _ := v.(string)
		if contains(s.failed, state) {
			msg, _ := obj["message"].(string)
			if msg == "" {
				msg, _ = obj["errorMessage"].(string)
			}
			return false, fmt.Errorf("it is in state %s %s", state, msg)
		}
		return contains(s.ready, state), nil
	})
}

func waitGone(ctx context.Context, get func() (map[string]any, error)) error {
	return pollUntil(ctx, func() (bool, error) {
		_, err := get()
		if client.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

func ids(kv ...any) Values {
	out := Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1]
	}
	return out
}

func str(obj map[string]any, path string) string {
	v, _ := lookup(obj, path)
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	}
	return ""
}

func set(body map[string]any, key string, v any) {
	switch x := v.(type) {
	case string:
		if x == "" {
			return
		}
	case int64:
		if x == 0 {
			return
		}
	case nil:
		return
	}
	body[key] = v
}

func strList(l []string) []any {
	out := make([]any, 0, len(l))
	for _, s := range l {
		out = append(out, s)
	}
	return out
}

func strMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func listFind(list map[string]any, field, key, value string) (map[string]any, error) {
	items, _ := list[field].([]any)
	for _, it := range items {
		if m, ok := it.(map[string]any); ok && str(m, key) == value {
			return m, nil
		}
	}
	return nil, errGone
}

func idAttr(path string) Attr {
	return Attr{Name: "id", Desc: "The id.", Kind: String, Computed: true, Stable: true, Path: path}
}

func createdAttr(path string) Attr {
	return Attr{Name: "created_at", Desc: "When it was created.", Kind: String, Computed: true, Stable: true, Path: path}
}

func merge(prior, plan Values) Values {
	out := Values{}
	for k, v := range prior {
		out[k] = v
	}
	for k, v := range plan {
		if !isUnknown(v) {
			out[k] = v
		}
	}
	return out
}

func fieldMask(paths ...string) string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		parts := strings.Split(p, ".")
		for i := range parts {
			parts[i] = camel(parts[i])
		}
		out = append(out, strings.Join(parts, "."))
	}
	return strings.Join(out, ",")
}
