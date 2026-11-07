---
page_title: "adatage_public_ip Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A public IPv4 address, optionally translated 1:1 (NAT) to a private address that an instance or a load balancer holds in a subnet.
---

# adatage_public_ip (Resource)

A public IPv4 address, optionally translated 1:1 (NAT) to a private address that an instance or a load balancer holds in a subnet. Set subnet_id and private_ipv4 together to attach it.

## Example Usage

```terraform
resource "adatage_public_ip" "web" {
  name        = "web"
  region_code = "eu-central-01"

  # translated 1:1 to a private address of an instance (or of a load balancer) in the subnet
  subnet_id    = adatage_subnet.app.id
  private_ipv4 = adatage_instance.web.primary_ipv4
}
```

## Schema

### Required

- `name` (String) 
- `region_code` (String) The region, for example eu-central-01. (changing it replaces the resource)

### Optional

- `private_ipv4` (String) The private address the public address is translated to (required with subnet_id): the address of an instance interface or of a load balancer.
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
