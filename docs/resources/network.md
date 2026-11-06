---
page_title: "adatage_network Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A virtual private network (VPC).
---

# adatage_network (Resource)

A virtual private network (VPC). Subnets carve it up.

## Example Usage

```terraform
resource "adatage_network" "main" {
  name = "main"
}
```

## Schema

### Required

- `name` (String) The name of the network.

### Optional

- `ipv6_enabled` (Boolean) Whether the network has an IPv6 range. (changing it replaces the resource)
- `ipv6_ula` (String) The IPv6 unique local range, chosen when empty. (changing it replaces the resource)
- `mtu` (Number) The MTU of the network (default chosen by the platform). (changing it replaces the resource)

### Read-Only

- `created_at` (String) When it was created.
- `gateway_mac` (String) The MAC address of the gateways of the network.
- `id` (String) The id.
- `state` (String) pending, provisioning, ready, deleting or error.

## Import

```shell
terraform import adatage_network.example ID
```
