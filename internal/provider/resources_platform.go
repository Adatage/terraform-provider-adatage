package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
)

var clusterRunning = states{path: "state", ready: []string{"STATE_RUNNING"}, failed: []string{"STATE_ERROR"}}

var taintAttrs = []Attr{
	{Name: "key", Kind: String, Required: true},
	{Name: "value", Kind: String, Required: true},
	{Name: "effect", Kind: String, Required: true, Desc: "NoSchedule, PreferNoSchedule or NoExecute."},
}

func taints(l []Values) []any {
	out := []any{}
	for _, t := range l {
		out = append(out, map[string]any{"key": t.Str("key"), "value": t.Str("value"), "effect": t.Str("effect")})
	}
	return out
}

func poolBody(p Values) map[string]any {
	body := map[string]any{"name": p.Str("name"), "seriesCode": p.Str("series_code"), "vcpus": p.Int("vcpus"), "memoryMib": p.Int("memory_mib"), "size": p.Int("size")}
	set(body, "diskGib", p.Int("disk_gib"))
	set(body, "maxPods", p.Int("max_pods"))
	set(body, "minSize", p.Int("min_size"))
	set(body, "maxSize", p.Int("max_size"))
	if m := strMap(p.Map("labels")); len(m) > 0 {
		body["labels"] = m
	}
	if t := taints(p.Objs("taints")); len(t) > 0 {
		body["taints"] = t
	}
	return body
}

func poolSettings(p Values) map[string]any {
	return map[string]any{"labels": strMap(p.Map("labels")), "taints": taints(p.Objs("taints")), "minSize": p.Int("min_size"), "maxSize": p.Int("max_size")}
}

func kubernetesClusterResource() *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcKubernetes, "GetCluster", map[string]any{"clusterId": id})
		return obj, gone(err)
	}
	return &Resource{
		Name: "kubernetes_cluster",
		Desc: "A managed Kubernetes cluster with a highly available control plane. node_pool describes the first pool at creation; manage more pools (and later changes to the first one) with adatage_kubernetes_node_pool.",
		Attrs: []Attr{
			idAttr("clusterId"),
			{Name: "name", Kind: String, Required: true, ForceNew: true},
			{Name: "region_code", Kind: String, Required: true, ForceNew: true},
			{Name: "zone_code", Kind: String, Required: true, ForceNew: true},
			{Name: "subnet_id", Kind: String, Required: true, ForceNew: true, Desc: "The subnet of the nodes; pods get addresses from it."},
			{Name: "control_plane_count", Kind: Int, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "1 or 3."},
			{Name: "kubernetes_version", Kind: String, Optional: true, Computed: true, Desc: "Setting a newer version upgrades the cluster."},
			{Name: "node_pool", Kind: Object, Required: true, Local: true, Desc: "The first node pool; only used when the cluster is created.", Nested: []Attr{
				{Name: "name", Kind: String, Required: true},
				{Name: "series_code", Kind: String, Required: true},
				{Name: "vcpus", Kind: Int, Required: true},
				{Name: "memory_mib", Kind: Int, Required: true},
				{Name: "disk_gib", Kind: Int, Optional: true},
				{Name: "size", Kind: Int, Required: true},
				{Name: "max_pods", Kind: Int, Optional: true},
				{Name: "labels", Kind: StringMap, Optional: true},
			}},
			{Name: "endpoint", Kind: String, Computed: true, Stable: true, Desc: "The address of the Kubernetes API."},
			{Name: "state", Kind: String, Computed: true, Enum: "STATE_", Desc: "provisioning, running, error or deleting."},
			{Name: "message", Kind: String, Computed: true},
			{Name: "node_count", Kind: Int, Computed: true},
			{Name: "ready_node_count", Kind: Int, Computed: true},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"name": p.Str("name"), "regionCode": p.Str("region_code"), "zoneCode": p.Str("zone_code"), "subnetId": p.Str("subnet_id"), "nodePool": poolBody(p.Obj("node_pool"))}
			set(body, "controlPlaneCount", p.Int("control_plane_count"))
			obj, err := c.Call(ctx, svcKubernetes, "CreateCluster", body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "clusterId")
			err = waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, clusterRunning)
			if err == nil && p.Has("kubernetes_version") {
				cur, gerr := get(ctx, c, id)
				if gerr == nil && str(cur, "kubernetesVersion") != p.Str("kubernetes_version") {
					_, err = c.Call(ctx, svcKubernetes, "UpgradeCluster", map[string]any{"clusterId": id, "kubernetesVersion": p.Str("kubernetes_version")})
					if err == nil {
						time.Sleep(5 * time.Second)
						err = waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, clusterRunning)
					}
				}
			}
			return ids("id", id), err
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			if p.Has("kubernetes_version") && p.Str("kubernetes_version") != s.Str("kubernetes_version") {
				id := s.Str("id")
				if _, err := c.Call(ctx, svcKubernetes, "UpgradeCluster", map[string]any{"clusterId": id, "kubernetesVersion": p.Str("kubernetes_version")}); err != nil {
					return err
				}
				time.Sleep(5 * time.Second)
				return waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, clusterRunning)
			}
			return nil
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, svcKubernetes, "DeleteCluster", map[string]any{"clusterId": s.Str("id")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

func kubernetesNodePoolResource() *Resource {
	get := func(ctx context.Context, c *client.Client, cluster, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcKubernetes, "GetNodePool", map[string]any{"clusterId": cluster, "nodePoolId": id})
		return obj, gone(err)
	}
	return &Resource{
		Name:         "kubernetes_node_pool",
		Desc:         "A pool of identical nodes of a cluster. Import with cluster_id/node_pool_id.",
		ImportFields: []string{"cluster_id", "id"},
		Attrs: []Attr{
			idAttr("nodePoolId"),
			{Name: "cluster_id", Kind: String, Required: true, ForceNew: true},
			{Name: "name", Kind: String, Required: true, ForceNew: true},
			{Name: "series_code", Kind: String, Required: true, ForceNew: true},
			{Name: "vcpus", Kind: Int, Required: true, ForceNew: true},
			{Name: "memory_mib", Kind: Int, Required: true, ForceNew: true},
			{Name: "disk_gib", Kind: Int, Optional: true, Computed: true, Stable: true, ForceNew: true},
			{Name: "max_pods", Kind: Int, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "From 8 to 110, default 30."},
			{Name: "size", Kind: Int, Required: true, Desc: "The number of nodes."},
			{Name: "min_size", Kind: Int, Optional: true, Computed: true, Desc: "With max_size turns on the autoscaler."},
			{Name: "max_size", Kind: Int, Optional: true, Computed: true},
			{Name: "labels", Kind: StringMap, Optional: true, Computed: true},
			{Name: "taints", Kind: List, Optional: true, Computed: true, Nested: taintAttrs},
			{Name: "ready_nodes", Kind: Int, Computed: true},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := poolBody(p)
			body["clusterId"] = p.Str("cluster_id")
			obj, err := c.Call(ctx, svcKubernetes, "CreateNodePool", body)
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "nodePoolId")), nil
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("cluster_id"), s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			m := merge(s, p)
			_, err := c.Call(ctx, svcKubernetes, "UpdateNodePool", map[string]any{"clusterId": s.Str("cluster_id"), "nodePoolId": s.Str("id"), "size": m.Int("size"), "settings": poolSettings(m)})
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, svcKubernetes, "DeleteNodePool", map[string]any{"clusterId": s.Str("cluster_id"), "nodePoolId": s.Str("id")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("cluster_id"), s.Str("id")) })
		},
	}
}

func healthCheckResource() *Resource {
	read := func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
		list, err := c.Call(ctx, svcLB, "ListHealthChecks", nil)
		if err != nil {
			return nil, err
		}
		return listFind(list, "healthChecks", "healthCheckId", s.Str("id"))
	}
	fields := func(p Values) map[string]any {
		b := map[string]any{"protocol": Enum("PROTOCOL_", p.Str("protocol")), "port": p.Int("port")}
		set(b, "requestPath", p.Str("request_path"))
		set(b, "intervalSeconds", p.Int("interval_seconds"))
		set(b, "timeoutSeconds", p.Int("timeout_seconds"))
		set(b, "healthyThreshold", p.Int("healthy_threshold"))
		set(b, "unhealthyThreshold", p.Int("unhealthy_threshold"))
		return b
	}
	return &Resource{
		Name: "load_balancer_health_check",
		Desc: "A health check that a backend service uses to decide which backends receive traffic.",
		Attrs: []Attr{
			idAttr("healthCheckId"),
			{Name: "name", Kind: String, Required: true, ForceNew: true},
			{Name: "protocol", Kind: String, Required: true, Enum: "PROTOCOL_", Desc: "tcp or http."},
			{Name: "port", Kind: Int, Required: true},
			{Name: "request_path", Kind: String, Optional: true, Computed: true, Desc: "http only; / by default."},
			{Name: "interval_seconds", Kind: Int, Optional: true, Computed: true, Desc: "2 to 300, default 5."},
			{Name: "timeout_seconds", Kind: Int, Optional: true, Computed: true, Desc: "1 to the interval, default 3."},
			{Name: "healthy_threshold", Kind: Int, Optional: true, Computed: true, Desc: "1 to 10, default 2."},
			{Name: "unhealthy_threshold", Kind: Int, Optional: true, Computed: true, Desc: "1 to 10, default 3."},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := fields(p)
			body["name"] = p.Str("name")
			obj, err := c.Call(ctx, svcLB, "CreateHealthCheck", body)
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "healthCheckId")), nil
		},
		Read: read,
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			body := fields(merge(s, p))
			body["healthCheckId"] = s.Str("id")
			_, err := c.Call(ctx, svcLB, "UpdateHealthCheck", body)
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcLB, "DeleteHealthCheck", map[string]any{"healthCheckId": s.Str("id")})
			return gone(err)
		},
	}
}

func backendServiceResource() *Resource {
	read := func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
		obj, err := c.Call(ctx, svcLB, "GetBackendService", map[string]any{"backendServiceId": s.Str("id")})
		return obj, gone(err)
	}
	backends := func(p Values) []any {
		out := []any{}
		for _, b := range p.Objs("backends") {
			m := map[string]any{"instanceId": b.Str("instance_id")}
			set(m, "nicId", b.Str("nic_id"))
			if b.Has("weight") {
				m["weight"] = b.Int("weight")
			}
			out = append(out, m)
		}
		return out
	}
	return &Resource{
		Name: "load_balancer_backend_service",
		Desc: "The instances a forwarding rule sends traffic to, with a health check.",
		Attrs: []Attr{
			idAttr("backendServiceId"),
			{Name: "name", Kind: String, Required: true, ForceNew: true},
			{Name: "health_check_id", Kind: String, Optional: true, Computed: true},
			{Name: "affinity", Kind: String, Optional: true, Computed: true, Enum: "AFFINITY_", Desc: "none or client_ip."},
			{Name: "backends", Kind: List, Optional: true, Computed: true, Desc: "The instances (and interfaces) that get traffic.", Nested: []Attr{
				{Name: "instance_id", Kind: String, Required: true, Path: "backend.instanceId"},
				{Name: "nic_id", Kind: String, Optional: true, Computed: true, Path: "backend.nicId"},
				{Name: "weight", Kind: Int, Optional: true, Computed: true, Path: "backend.weight"},
			}},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"name": p.Str("name"), "backends": backends(p)}
			set(body, "healthCheckId", p.Str("health_check_id"))
			set(body, "affinity", Enum("AFFINITY_", p.Str("affinity")))
			obj, err := c.Call(ctx, svcLB, "CreateBackendService", body)
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "backendServiceId")), nil
		},
		Read: read,
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			m := merge(s, p)
			body := map[string]any{"backendServiceId": s.Str("id"), "healthCheckId": m.Str("health_check_id"), "affinity": Enum("AFFINITY_", m.Str("affinity")), "backends": backends(m)}
			_, err := c.Call(ctx, svcLB, "UpdateBackendService", body)
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcLB, "DeleteBackendService", map[string]any{"backendServiceId": s.Str("id")})
			return gone(err)
		},
	}
}

func forwardingRuleResource() *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcLB, "GetForwardingRule", map[string]any{"forwardingRuleId": id})
		return obj, gone(err)
	}
	return &Resource{
		Name: "load_balancer_forwarding_rule",
		Desc: "A layer 4 load balancer: a persistent private address in a subnet that forwards a port range to a backend service. Attach public addresses to its private address with adatage_public_ip.",
		Attrs: []Attr{
			idAttr("forwardingRuleId"),
			{Name: "name", Kind: String, Required: true, ForceNew: true},
			{Name: "region_code", Kind: String, Required: true, ForceNew: true},
			{Name: "subnet_id", Kind: String, Required: true, ForceNew: true},
			{Name: "address", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "The private address; chosen in the subnet when empty. It stays."},
			{Name: "protocol", Kind: String, Required: true, ForceNew: true, Enum: "LOAD_BALANCER_PROTOCOL_", Desc: "tcp or udp."},
			{Name: "port_from", Kind: Int, Required: true, ForceNew: true},
			{Name: "port_to", Kind: Int, Optional: true, Computed: true, ForceNew: true, Desc: "The last port; only port_from when empty."},
			{Name: "backend_service_id", Kind: String, Required: true, ForceNew: true},
			{Name: "state", Kind: String, Computed: true, Enum: "STATE_"},
			{Name: "public_addresses", Kind: StringList, Computed: true},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"name": p.Str("name"), "regionCode": p.Str("region_code"), "subnetId": p.Str("subnet_id"), "protocol": Enum("LOAD_BALANCER_PROTOCOL_", p.Str("protocol")),
				"portFrom": p.Int("port_from"), "backendServiceId": p.Str("backend_service_id")}
			set(body, "address", p.Str("address"))
			set(body, "portTo", p.Int("port_to"))
			obj, err := c.Call(ctx, svcLB, "CreateForwardingRule", body)
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "forwardingRuleId")), nil
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, svcLB, "DeleteForwardingRule", map[string]any{"forwardingRuleId": s.Str("id")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

func iamRoleResource() *Resource {
	read := func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
		list, err := c.Call(ctx, svcIam, "ListRoles", nil)
		if err != nil {
			return nil, err
		}
		return listFind(list, "roles", "roleId", s.Str("id"))
	}
	return &Resource{
		Name: "iam_role",
		Desc: "A custom role: a named set of permissions that can be given to members and service accounts.",
		Attrs: []Attr{
			idAttr("roleId"),
			{Name: "name", Kind: String, Required: true, Desc: "The display name; the id is made from it."},
			{Name: "permissions", Kind: StringList, Required: true, Desc: "Permission names such as compute.instances.list."},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			obj, err := c.Call(ctx, svcIam, "CreateRole", map[string]any{"name": p.Str("name"), "permissions": strList(p.Strings("permissions"))})
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "roleId")), nil
		},
		Read: read,
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			_, err := c.Call(ctx, svcIam, "UpdateRole", map[string]any{"roleId": s.Str("id"), "name": p.Str("name"), "permissions": strList(p.Strings("permissions"))})
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcIam, "DeleteRole", map[string]any{"roleId": s.Str("id")})
			return gone(err)
		},
	}
}

func serviceAccountResource() *Resource {
	read := func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
		obj, err := c.Call(ctx, svcAccounts, "GetServiceAccount", map[string]any{"serviceAccountId": s.Str("id")})
		return obj, gone(err)
	}
	return &Resource{
		Name: "service_account",
		Desc: "A service account: an identity for programs that holds one role in the project.",
		Attrs: []Attr{
			idAttr("serviceAccountId"),
			{Name: "name", Kind: String, Required: true, ForceNew: true, Desc: "Lowercase letters, digits and hyphens."},
			{Name: "display_name", Kind: String, Optional: true, Computed: true},
			{Name: "description", Kind: String, Optional: true, Computed: true},
			{Name: "role", Kind: String, Required: true, Desc: "The id of the role it holds in the project."},
			{Name: "disabled", Kind: Bool, Optional: true, Computed: true},
			createdAttr("createTime"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"name": p.Str("name"), "role": p.Str("role")}
			set(body, "displayName", p.Str("display_name"))
			set(body, "description", p.Str("description"))
			obj, err := c.Call(ctx, svcAccounts, "CreateServiceAccount", body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "serviceAccountId")
			if p.Bool("disabled") {
				_, err = c.Call(ctx, svcAccounts, "UpdateServiceAccount", map[string]any{"serviceAccountId": id, "disabled": true, "updateMask": fieldMask("disabled")})
			}
			return ids("id", id), err
		},
		Read: read,
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			m := merge(s, p)
			_, err := c.Call(ctx, svcAccounts, "UpdateServiceAccount", map[string]any{"serviceAccountId": s.Str("id"), "displayName": m.Str("display_name"), "description": m.Str("description"),
				"role": m.Str("role"), "disabled": m.Bool("disabled"), "updateMask": fieldMask("display_name", "description", "role", "disabled")})
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcAccounts, "DeleteServiceAccount", map[string]any{"serviceAccountId": s.Str("id")})
			return gone(err)
		},
	}
}

func serviceAccountKeyResource() *Resource {
	return &Resource{
		Name:         "service_account_key",
		Desc:         "A key of a service account. The credentials (the file to give to the provider, Terraform or a script) are only known when the key is created and are kept, sensitive, in the state.",
		ImportFields: []string{"service_account_id", "id"},
		Attrs: []Attr{
			idAttr("keyId"),
			{Name: "service_account_id", Kind: String, Required: true, ForceNew: true},
			{Name: "description", Kind: String, Optional: true, Computed: true, ForceNew: true},
			{Name: "expire_time", Kind: String, Optional: true, ForceNew: true, Local: true, Desc: "RFC 3339 time after which the key stops working; it does not expire when empty."},
			{Name: "credentials", Kind: String, Computed: true, Sensitive: true, Local: true, Desc: "The credentials file (JSON)."},
			createdAttr("createTime"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"serviceAccountId": p.Str("service_account_id")}
			set(body, "description", p.Str("description"))
			if p.Has("expire_time") {
				if _, err := time.Parse(time.RFC3339, p.Str("expire_time")); err != nil {
					return nil, fmt.Errorf("expire_time: %w", err)
				}
				body["expireTime"] = p.Str("expire_time")
			}
			obj, err := c.Call(ctx, svcAccounts, "CreateServiceAccountKey", body)
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "key.keyId"), "credentials", str(obj, "credentials")), nil
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			list, err := c.Call(ctx, svcAccounts, "ListServiceAccountKeys", map[string]any{"serviceAccountId": s.Str("service_account_id")})
			if err != nil {
				return nil, gone(err)
			}
			return listFind(list, "keys", "keyId", s.Str("id"))
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcAccounts, "DeleteServiceAccountKey", map[string]any{"serviceAccountId": s.Str("service_account_id"), "keyId": s.Str("id")})
			return gone(err)
		},
	}
}

func allResources() []*Resource {
	return []*Resource{
		sshKeyResource(), networkResource(), subnetResource(), firewallRuleResource(), publicIPResource(),
		diskResource(), instanceResource(), templateResource(), snapshotResource(),
		kubernetesClusterResource(), kubernetesNodePoolResource(),
		healthCheckResource(), backendServiceResource(), forwardingRuleResource(),
		databaseResource(), databaseBackupResource(),
		iamRoleResource(), serviceAccountResource(), serviceAccountKeyResource(),
	}
}
