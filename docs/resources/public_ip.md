---
page_title: "adatage_public_ip Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A public IPv4 address, optionally translated 1:1 (NAT) to a private address in a subnet.
---

# adatage_public_ip (Resource)

A public IPv4 address, optionally translated 1:1 (NAT) to a private address in a subnet. Set subnet_id (and optionally private_ipv4) to attach it; load balancers and Kubernetes use the same mechanism.

## Example Usage

```terraform
resource "adatage_public_ip" "web" {
  name        = "web"
  region_code = "eu-central-01"
  subnet_id   = adatage_subnet.app.id # translated 1:1 to a private address of the subnet
}
```

## Schema

### Required

- `name` (String) 
- `region_code` (String) The region, for example eu-central-01. (changing it replaces the resource)

### Optional

- `private_ipv4` (String) The private address; chosen in the subnet when empty.
- `subnet_id` (String) The subnet of the private address the public address is translated to.

### Read-Only

- `address` (String) The public address.
- `created_at` (String) When it was created.
- `id` (String) The id.
- `network_id` (String) 
- `state` (String) unassociated, pending, active or balanced.

## Import

```shell
terraform import adatage_public_ip.example ID
```
