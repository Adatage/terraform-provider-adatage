package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSchemasAreValid(t *testing.T) {
	ctx := context.Background()
	for _, spec := range allResources() {
		r := &genericResource{spec: spec}
		resp := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, resp)
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("resource %s: %v", spec.Name, diags)
		}
		if _, ok := resp.Schema.Attributes["id"]; !ok {
			t.Errorf("resource %s has no id", spec.Name)
		}
		for _, f := range spec.importFields() {
			if _, ok := resp.Schema.Attributes[f]; !ok && spec.ImportParse == nil {
				t.Errorf("resource %s: import field %s is not an attribute", spec.Name, f)
			}
		}
		if spec.Create == nil || spec.Read == nil || spec.Delete == nil {
			t.Errorf("resource %s lacks Create, Read or Delete", spec.Name)
		}
	}
	for _, spec := range allDataSources() {
		d := &genericData{spec: spec}
		resp := &datasource.SchemaResponse{}
		d.Schema(ctx, datasource.SchemaRequest{}, resp)
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("data source %s: %v", spec.Name, diags)
		}
	}
}

// Every attribute of a resource that cannot change in place must be replaced, and every other one needs an Update to work.
func TestUpdatableAttributesHaveUpdate(t *testing.T) {
	for _, spec := range allResources() {
		if spec.Update != nil {
			continue
		}
		for _, a := range spec.Attrs {
			if (a.Required || a.Optional) && !a.ForceNew && !a.Local {
				t.Errorf("%s.%s can change but the resource has no Update", spec.Name, a.Name)
			}
		}
	}
}

func TestReadObject(t *testing.T) {
	attrs := []Attr{
		{Name: "id", Kind: String, Computed: true, Path: "networkId"},
		{Name: "name", Kind: String, Required: true, Path: "metadata.name"},
		{Name: "mtu", Kind: Int, Optional: true, Computed: true},
		{Name: "state", Kind: String, Computed: true, Enum: "PHASE_STATE_"},
		{Name: "ipv6_enabled", Kind: Bool, Optional: true, Computed: true},
		{Name: "note", Kind: String, Optional: true},
		{Name: "tags", Kind: StringList, Optional: true, Computed: true},
		{Name: "secret", Kind: String, Computed: true, Local: true},
		{Name: "pools", Kind: List, Computed: true, Nested: []Attr{{Name: "instance_id", Kind: String, Path: "backend.instanceId"}, {Name: "weight", Kind: Int, Path: "backend.weight"}}},
	}
	obj := map[string]any{
		"networkId": "9007199254740993", "metadata": map[string]any{"name": "main"}, "mtu": float64(1500), "state": "PHASE_STATE_READY",
		"note":  "ignored because the plan had none",
		"pools": []any{map[string]any{"backend": map[string]any{"instanceId": "5", "weight": float64(3)}}},
	}
	got := readObject(attrs, obj, Values{"secret": "kept", "tags": unknown})
	if got.Str("id") != "9007199254740993" || got.Str("name") != "main" || got.Int("mtu") != 1500 || got.Str("state") != "ready" {
		t.Fatalf("%v", got)
	}
	if got["ipv6_enabled"] != false || got["note"] != nil || got.Str("secret") != "kept" {
		t.Fatalf("zero values, null optional and local values: %v", got)
	}
	if l, ok := got["tags"].([]any); !ok || len(l) != 0 {
		t.Fatalf("a missing list is empty: %v", got["tags"])
	}
	pools := got.Objs("pools")
	if len(pools) != 1 || pools[0].Str("instance_id") != "5" || pools[0].Int("weight") != 3 {
		t.Fatalf("nested: %v", got["pools"])
	}
	raw, err := encode(attrs, got)
	if err != nil {
		t.Fatal(err)
	}
	back, err := decode(attrs, raw)
	if err != nil || back.Str("name") != "main" || back.Int("mtu") != 1500 {
		t.Fatalf("round trip: %v %v", back, err)
	}
}

func TestEnum(t *testing.T) {
	if Enum("IP_TYPE_", "dual_stack") != "IP_TYPE_DUAL_STACK" || Enum("X_", "") != "" || fromEnum("IP_TYPE_", "IP_TYPE_IPV4_ONLY") != "ipv4_only" {
		t.Fatal("enum spelling")
	}
}

// fakeAPI answers the calls of the network lifecycle.
type fakeAPI struct {
	mu      sync.Mutex
	exists  bool
	calls   []string
	project string
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	f.calls = append(f.calls, method)
	f.project = r.Header.Get("X-Project-Id")
	if r.Header.Get("Authorization") != "Bearer token" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"unauthenticated","message":"no"}`))
		return
	}
	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)
	network := map[string]any{"networkId": "7", "metadata": map[string]any{"name": body["name"], "createdAt": "2026-11-06T10:00:00Z"}, "mtu": 1500, "state": "PHASE_STATE_READY", "gatewayMac": "02:00:00:00:00:01"}
	switch method {
	case "CreateNetwork":
		f.exists = true
		network["metadata"].(map[string]any)["name"] = body["name"]
		json.NewEncoder(w).Encode(network)
	case "GetNetwork":
		if !f.exists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":"not_found","message":"no such network"}`))
			return
		}
		network["metadata"].(map[string]any)["name"] = "main"
		json.NewEncoder(w).Encode(network)
	case "DeleteNetwork":
		f.exists = false
		w.Write([]byte(`{}`))
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"unimplemented","message":"` + method + `"}`))
	}
}

func TestNetworkLifecycle(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()
	c, err := client.New(srv.URL, "42", nil, "token")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	r := &genericResource{spec: networkResource(), c: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	typ := objectType(r.spec.Attrs)

	plan, err := fromGo(typ, map[string]any{"id": unknown, "name": "main", "mtu": unknown, "ipv6_enabled": unknown, "ipv6_ula": unknown, "gateway_mac": unknown, "state": unknown, "created_at": unknown})
	if err != nil {
		t.Fatal(err)
	}
	created := &resource.CreateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(typ, nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: plan}}, created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	state, err := decode(r.spec.Attrs, created.State.Raw)
	if err != nil || state.Str("id") != "7" || state.Str("state") != "ready" || state.Int("mtu") != 1500 || state.Str("gateway_mac") == "" {
		t.Fatalf("state after create: %v %v", state, err)
	}
	if api.project != "42" {
		t.Fatalf("project header %q", api.project)
	}

	read := &resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, read)
	if read.Diagnostics.HasError() || read.State.Raw.IsNull() {
		t.Fatalf("read: %v", read.Diagnostics)
	}

	r.Delete(ctx, resource.DeleteRequest{State: created.State}, &resource.DeleteResponse{})
	gone := &resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, gone)
	if !gone.State.Raw.IsNull() {
		t.Fatal("a deleted network must leave the state")
	}
}

func TestClientErrorsAndSignIn(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()
	c, _ := client.New(srv.URL, "1", nil, "wrong")
	_, err := c.Call(context.Background(), "x.Y", "GetNetwork", nil)
	var ae *client.Error
	if err == nil || !strings.Contains(err.Error(), "unauthenticated") {
		t.Fatalf("%v %v", err, ae)
	}
	_, err = c.Call(context.Background(), svcNetwork, "Nope", nil)
	if err == nil {
		t.Fatal("unauthorized stays an error")
	}
}

// The framework's server validates the whole schema the way Terraform's protocol sees it.
func TestProviderServesItsSchema(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("%s: %s", d.Summary, d.Detail)
	}
	if len(resp.ResourceSchemas) != len(allResources()) || len(resp.DataSourceSchemas) != len(allDataSources()) {
		t.Fatalf("%d resources and %d data sources served", len(resp.ResourceSchemas), len(resp.DataSourceSchemas))
	}
	if _, ok := resp.ResourceSchemas["adatage_instance"]; !ok {
		t.Fatal("adatage_instance is missing")
	}
}

// Plans the creation of an instance and a cluster through the protocol: nested attributes, unknown computed values and replacement.
func TestPlanCreate(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	plan := func(name string, spec *Resource, config Values) *tfprotov6.PlanResourceChangeResponse {
		typ := schemas.ResourceSchemas[name].ValueType()
		obj := typ.(tftypes.Object)
		cfg := Values{}
		for _, a := range spec.Attrs {
			cfg[a.Name] = config[a.Name]
		}
		val, err := fromGo(obj, map[string]any(cfg))
		if err != nil {
			t.Fatal(err)
		}
		dv, err := tfprotov6.NewDynamicValue(obj, val)
		if err != nil {
			t.Fatal(err)
		}
		null, _ := tfprotov6.NewDynamicValue(obj, tftypes.NewValue(obj, nil))
		resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: name, PriorState: &null, ProposedNewState: &dv, Config: &dv})
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range resp.Diagnostics {
			t.Errorf("%s: %s: %s", name, d.Summary, d.Detail)
		}
		return resp
	}
	plan("adatage_instance", instanceResource(), Values{
		"name": "web", "region_code": "r", "zone_code": "a", "series_code": "general", "vcpus": int64(2), "memory_mib": int64(2048),
		"disks": []any{map[string]any{"is_boot": true, "name": "boot", "size_gib": int64(20)}},
		"nics":  []any{map[string]any{"subnet_id": "5"}},
	})
	plan("adatage_kubernetes_cluster", kubernetesClusterResource(), Values{
		"name": "k", "region_code": "r", "zone_code": "a", "subnet_id": "5",
		"node_pool": map[string]any{"name": "p", "series_code": "general", "vcpus": int64(2), "memory_mib": int64(2048), "size": int64(3)},
	})
	plan("adatage_database_instance", databaseResource(), Values{
		"name": "db", "engine": "postgres", "region_code": "r", "zone_code": "a", "subnet_id": "5", "vcpus": int64(2), "memory_mib": int64(2048), "disk_gib": int64(20),
		"allowed_cidrs": []any{"203.0.113.0/24"}, "labels": map[string]any{"team": "a"},
	})
	plan("adatage_load_balancer_backend_service", backendServiceResource(), Values{"name": "b", "backends": []any{map[string]any{"instance_id": "9"}}})
	plan("adatage_kubernetes_node_pool", kubernetesNodePoolResource(), Values{"cluster_id": "1", "name": "p", "series_code": "g", "vcpus": int64(2), "memory_mib": int64(2048), "size": int64(1),
		"taints": []any{map[string]any{"key": "k", "value": "v", "effect": "NoSchedule"}}})
}

func TestKeepSpelling(t *testing.T) {
	text := Attr{Name: "k", Kind: String}
	list := Attr{Name: "l", Kind: StringList}
	if got := keepSpelling(text, "ssh-ed25519 AAA k\n", "ssh-ed25519 AAA k"); got != "ssh-ed25519 AAA k\n" {
		t.Fatalf("whitespace: %q", got)
	}
	if got := keepSpelling(text, "a", "b"); got != "b" {
		t.Fatalf("a real change stays: %q", got)
	}
	if got := keepSpelling(list, []any{"b", "a"}, []any{"a", "b"}); got.([]any)[0] != "b" {
		t.Fatalf("the order of the plan stays: %v", got)
	}
	if got := keepSpelling(list, []any{"b", "a"}, []any{"a", "c"}); got.([]any)[1] != "c" {
		t.Fatalf("a real change stays: %v", got)
	}
}

func TestFieldMask(t *testing.T) {
	if got := fieldMask("name", "memory_mib", "spec.disk_gib"); got != "name,memoryMib,spec.diskGib" {
		t.Fatalf("field mask: %q", got)
	}
}
