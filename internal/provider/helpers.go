package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adatage/terraform-provider-adatage/internal/client"
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

// Ready and failed states are the API's own spellings.
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

// pollUntil calls fn every few seconds until it says it is done.
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

// waitState waits until the object's state is one of the ready values.
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

// waitGone waits until the object cannot be found any more.
func waitGone(ctx context.Context, get func() (map[string]any, error)) error {
	return pollUntil(ctx, func() (bool, error) {
		_, err := get()
		if client.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// lastSegment is the id at the end of a name such as projects/1/instances/2.
func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// ids is the shorthand for the Values a Create returns.
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

// set puts the Terraform value of an attribute into a request when it has one.
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

// listFind finds the element of a list answer whose key has the value.
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

// merge is the prior values with the known ones of the plan on top: what an update sends for attributes the plan leaves open.
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
