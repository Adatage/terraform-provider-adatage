---
page_title: "adatage_subnet Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A subnet of a network.
---

# adatage_subnet (Resource)

A subnet of a network. Import with network_id/subnet_id.

## Example Usage

```terraform
resource "adatage_subnet" "app" {
  network_id = adatage_network.main.id
  name       = "app"
  ip_type    = "ipv4_only"
  ipv4_cidr  = "10.20.0.0/24"
}
```

## Schema

### Required

- `ip_type` (String) ipv4_only, ipv6_only or dual_stack. (changing it replaces the resource)
- `name` (String) 
- `network_id` (String) The network the subnet belongs to. (changing it replaces the resource)

### Optional

- `ipv4_cidr` (String) The IPv4 range, for example 10.0.1.0/24. (changing it replaces the resource)
- `ipv6_cidr` (String) (changing it replaces the resource)

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.
- `state` (String) 

## Import

```shell
terraform import adatage_subnet.example NETWORK_ID/ID
```
