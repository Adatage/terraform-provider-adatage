package provider

import (
	"context"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
)

var networkReady = states{path: "state", ready: []string{"PHASE_STATE_READY"}, failed: []string{"PHASE_STATE_ERROR"}}

func sshKeyResource() *Resource {
	return &Resource{
		Name: "ssh_key",
		Desc: "A public SSH key of the project that instances can be given.",
		Attrs: []Attr{
			{Name: "id", Kind: String, Computed: true, Stable: true, Path: "name", Desc: "The name of the key."},
			{Name: "name", Kind: String, Required: true, ForceNew: true, Desc: "The name of the key."},
			{Name: "public_key", Kind: String, Required: true, ForceNew: true, Desc: "The public key in the authorized_keys format."},
			{Name: "key_type", Kind: String, Computed: true, Stable: true, Desc: "ssh-ed25519, ssh-rsa and so on."},
			{Name: "fingerprint", Kind: String, Computed: true, Stable: true, Desc: "SHA256:... as ssh-keygen prints it."},
			{Name: "comment", Kind: String, Computed: true, Stable: true},
			createdAttr("createTime"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			if _, err := c.Call(ctx, svcSshKey, "CreateSshKey", map[string]any{"name": p.Str("name"), "publicKey": p.Str("public_key")}); err != nil {
				return nil, err
			}
			return ids("id", p.Str("name")), nil
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			obj, err := c.Call(ctx, svcSshKey, "GetSshKey", map[string]any{"name": s.Str("id")})
			return obj, gone(err)
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcSshKey, "DeleteSshKey", map[string]any{"name": s.Str("id")})
			return gone(err)
		},
	}
}

func networkResource() *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcNetwork, "GetNetwork", map[string]any{"networkId": id})
		return obj, gone(err)
	}
	return &Resource{
		Name: "network",
		Desc: "A virtual private network (VPC). Subnets carve it up.",
		Attrs: []Attr{
			idAttr("networkId"),
			{Name: "name", Kind: String, Required: true, Path: "metadata.name", Desc: "The name of the network."},
			{Name: "mtu", Kind: Int, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "The MTU of the network (default chosen by the platform)."},
			{Name: "ipv6_enabled", Kind: Bool, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "Whether the network has an IPv6 range."},
			{Name: "ipv6_ula", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "The IPv6 unique local range, chosen when empty."},
			{Name: "gateway_mac", Kind: String, Computed: true, Stable: true, Desc: "The MAC address of the gateways of the network."},
			{Name: "state", Kind: String, Computed: true, Enum: "PHASE_STATE_", Desc: "pending, provisioning, ready, deleting or error."},
			createdAttr("metadata.createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"name": p.Str("name")}
			set(body, "mtu", p.Int("mtu"))
			if p.Has("ipv6_enabled") {
				body["ipv6Enabled"] = p.Bool("ipv6_enabled")
			}
			set(body, "ipv6Ula", p.Str("ipv6_ula"))
			obj, err := c.Call(ctx, svcNetwork, "CreateNetwork", body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "networkId")
			if err := waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, networkReady); err != nil {
				return ids("id", id), err
			}
			return ids("id", id), nil
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			_, err := c.Call(ctx, svcNetwork, "UpdateNetwork", map[string]any{"networkId": s.Str("id"), "name": p.Str("name")})
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, svcNetwork, "DeleteNetwork", map[string]any{"networkId": s.Str("id")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

func subnetResource() *Resource {
	get := func(ctx context.Context, c *client.Client, network, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcNetwork, "GetSubnet", map[string]any{"networkId": network, "subnetId": id})
		return obj, gone(err)
	}
	return &Resource{
		Name:         "subnet",
		Desc:         "A subnet of a network. Import with network_id/subnet_id.",
		ImportFields: []string{"network_id", "id"},
		Attrs: []Attr{
			idAttr("subnetId"),
			{Name: "network_id", Kind: String, Required: true, ForceNew: true, Desc: "The network the subnet belongs to."},
			{Name: "name", Kind: String, Required: true, Path: "metadata.name"},
			{Name: "ip_type", Kind: String, Required: true, ForceNew: true, Path: "type", Enum: "IP_TYPE_", Desc: "ipv4_only, ipv6_only or dual_stack."},
			{Name: "ipv4_cidr", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "The IPv4 range, for example 10.0.1.0/24."},
			{Name: "ipv6_cidr", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true},
			{Name: "state", Kind: String, Computed: true, Enum: "PHASE_STATE_"},
			createdAttr("metadata.createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"networkId": p.Str("network_id"), "name": p.Str("name"), "type": Enum("IP_TYPE_", p.Str("ip_type"))}
			set(body, "ipv4Cidr", p.Str("ipv4_cidr"))
			set(body, "ipv6Cidr", p.Str("ipv6_cidr"))
			obj, err := c.Call(ctx, svcNetwork, "CreateSubnet", body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "subnetId")
			err = waitState(ctx, func() (map[string]any, error) { return get(ctx, c, p.Str("network_id"), id) }, networkReady)
			return ids("id", id), err
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("network_id"), s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			_, err := c.Call(ctx, svcNetwork, "UpdateSubnet", map[string]any{"subnetId": s.Str("id"), "name": p.Str("name")})
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, svcNetwork, "DeleteSubnet", map[string]any{"subnetId": s.Str("id")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("network_id"), s.Str("id")) })
		},
	}
}

func firewallRuleResource() *Resource {
	read := func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
		list, err := c.Call(ctx, svcNetwork, "ListRules", map[string]any{"networkId": s.Str("network_id")})
		if err != nil {
			return nil, gone(err)
		}
		return listFind(list, "rules", "ruleId", s.Str("id"))
	}
	rule := func(p Values, id string) map[string]any {
		r := map[string]any{
			"direction": Enum("DIRECTION_", p.Str("direction")), "action": Enum("ACTION_", p.Str("action")), "protocol": Enum("PROTOCOL_", p.Str("protocol")),
			"remoteCidr": p.Str("remote_cidr"), "portStart": p.Int("port_start"), "portEnd": p.Int("port_end"), "priority": p.Int("priority"), "name": p.Str("name"),
		}
		if id != "" {
			r["ruleId"] = id
		}
		return r
	}
	return &Resource{
		Name:         "firewall_rule",
		Desc:         "A firewall rule of a network. Import with network_id/rule_id.",
		ImportFields: []string{"network_id", "id"},
		Attrs: []Attr{
			idAttr("ruleId"),
			{Name: "network_id", Kind: String, Required: true, ForceNew: true},
			{Name: "name", Kind: String, Optional: true, Computed: true},
			{Name: "direction", Kind: String, Required: true, Enum: "DIRECTION_", Desc: "ingress or egress."},
			{Name: "action", Kind: String, Required: true, Enum: "ACTION_", Desc: "allow or deny."},
			{Name: "protocol", Kind: String, Required: true, Enum: "PROTOCOL_", Desc: "any, tcp, udp, icmp or icmpv6."},
			{Name: "remote_cidr", Kind: String, Required: true, Desc: "The remote address range, 0.0.0.0/0 for everywhere."},
			{Name: "port_start", Kind: Int, Optional: true, Computed: true, Desc: "First port (tcp and udp)."},
			{Name: "port_end", Kind: Int, Optional: true, Computed: true, Desc: "Last port; the same as port_start for one port."},
			{Name: "priority", Kind: Int, Optional: true, Computed: true, Desc: "Lower numbers are evaluated first."},
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			obj, err := c.Call(ctx, svcNetwork, "CreateRule", map[string]any{"networkId": p.Str("network_id"), "rule": rule(p, "")})
			if err != nil {
				return nil, err
			}
			return ids("id", str(obj, "ruleId")), nil
		},
		Read: read,
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			_, err := c.Call(ctx, svcNetwork, "UpdateRule", map[string]any{"networkId": s.Str("network_id"), "rule": rule(merge(s, p), s.Str("id"))})
			return err
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcNetwork, "DeleteRule", map[string]any{"networkId": s.Str("network_id"), "ruleId": s.Str("id")})
			return gone(err)
		},
	}
}

func publicIPResource() *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcNetwork, "GetPublicIp", map[string]any{"publicIpId": id})
		return obj, gone(err)
	}
	nat := func(ctx context.Context, c *client.Client, id string, p Values) error {
		if p.Str("subnet_id") == "" {
			return nil
		}
		body := map[string]any{"publicIpId": id, "subnetId": p.Str("subnet_id")}
		set(body, "privateIpv4", p.Str("private_ipv4"))
		_, err := c.Call(ctx, svcNetwork, "AddPublicIpNat", body)
		return err
	}
	return &Resource{
		Name: "public_ip",
		Desc: "A public IPv4 address, optionally translated 1:1 (NAT) to a private address in a subnet. Set subnet_id (and optionally private_ipv4) to attach it; load balancers and Kubernetes use the same mechanism.",
		Attrs: []Attr{
			idAttr("publicIpId"),
			{Name: "name", Kind: String, Required: true},
			{Name: "region_code", Kind: String, Required: true, ForceNew: true, Desc: "The region, for example eu-central-01."},
			{Name: "address", Kind: String, Computed: true, Stable: true, Desc: "The public address."},
			{Name: "subnet_id", Kind: String, Optional: true, Desc: "The subnet of the private address the public address is translated to."},
			{Name: "private_ipv4", Kind: String, Optional: true, Computed: true, Desc: "The private address; chosen in the subnet when empty."},
			{Name: "network_id", Kind: String, Computed: true},
			{Name: "state", Kind: String, Computed: true, Enum: "STATE_", Desc: "unassociated, pending, active or balanced."},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			obj, err := c.Call(ctx, svcNetwork, "CreatePublicIp", map[string]any{"name": p.Str("name"), "regionCode": p.Str("region_code")})
			if err != nil {
				return nil, err
			}
			id := str(obj, "publicIpId")
			return ids("id", id), nat(ctx, c, id, p)
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			id := s.Str("id")
			if p.Str("name") != s.Str("name") {
				if _, err := c.Call(ctx, svcNetwork, "UpdatePublicIp", map[string]any{"publicIpId": id, "name": p.Str("name")}); err != nil {
					return err
				}
			}
			if p.Str("subnet_id") != s.Str("subnet_id") || (p.Has("private_ipv4") && p.Str("private_ipv4") != s.Str("private_ipv4")) {
				if s.Str("subnet_id") != "" {
					if _, err := c.Call(ctx, svcNetwork, "RemovePublicIpNat", map[string]any{"publicIpId": id}); err != nil {
						return err
					}
				}
				return nat(ctx, c, id, p)
			}
			return nil
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if s.Str("subnet_id") != "" {
				if _, err := c.Call(ctx, svcNetwork, "RemovePublicIpNat", map[string]any{"publicIpId": s.Str("id")}); err != nil && !client.IsNotFound(err) {
					return err
				}
			}
			_, err := c.Call(ctx, svcNetwork, "DeletePublicIp", map[string]any{"publicIpId": s.Str("id")})
			return gone(err)
		},
	}
}
