---
page_title: "adatage_firewall_rule Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A firewall rule of a network.
---

# adatage_firewall_rule (Resource)

A firewall rule of a network. Import with network_id/rule_id.

## Example Usage

```terraform
resource "adatage_firewall_rule" "ssh" {
  network_id  = adatage_network.main.id
  name        = "ssh"
  direction   = "ingress"
  action      = "allow"
  protocol    = "tcp"
  remote_cidr = "0.0.0.0/0"
  port_start  = 22
  port_end    = 22
  priority    = 100
}
```

## Schema

### Required

- `action` (String) allow or deny.
- `direction` (String) ingress or egress.
- `network_id` (String) (changing it replaces the resource)
- `protocol` (String) any, tcp, udp, icmp or icmpv6.
- `remote_cidr` (String) The remote address range, 0.0.0.0/0 for everywhere.

### Optional

- `name` (String) 
- `port_end` (Number) Last port; the same as port_start for one port.
- `port_start` (Number) First port (tcp and udp).
- `priority` (Number) Lower numbers are evaluated first.

### Read-Only

- `id` (String) The id.

## Import

```shell
terraform import adatage_firewall_rule.example NETWORK_ID/ID
```
