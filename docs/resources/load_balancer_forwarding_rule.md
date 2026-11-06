---
page_title: "adatage_load_balancer_forwarding_rule Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A layer 4 load balancer: a persistent private address in a subnet that forwards a port range to a backend service.
---

# adatage_load_balancer_forwarding_rule (Resource)

A layer 4 load balancer: a persistent private address in a subnet that forwards a port range to a backend service. Attach public addresses to its private address with adatage_public_ip.

## Example Usage

```terraform
resource "adatage_load_balancer_forwarding_rule" "web" {
  name               = "web"
  region_code        = "eu-central-01"
  subnet_id          = adatage_subnet.app.id
  protocol           = "tcp"
  port_from          = 80
  port_to            = 443
  backend_service_id = adatage_load_balancer_backend_service.web.id
}

# a public address 1:1 translated to the private address of the load balancer
resource "adatage_public_ip" "web_lb" {
  name         = "web-lb"
  region_code  = "eu-central-01"
  subnet_id    = adatage_subnet.app.id
  private_ipv4 = adatage_load_balancer_forwarding_rule.web.address
}
```

## Schema

### Required

- `backend_service_id` (String) (changing it replaces the resource)
- `name` (String) (changing it replaces the resource)
- `port_from` (Number) (changing it replaces the resource)
- `protocol` (String) tcp or udp. (changing it replaces the resource)
- `region_code` (String) (changing it replaces the resource)
- `subnet_id` (String) (changing it replaces the resource)

### Optional

- `address` (String) The private address; chosen in the subnet when empty. It stays. (changing it replaces the resource)
- `port_to` (Number) The last port; only port_from when empty. (changing it replaces the resource)

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.
- `public_addresses` (List of String) 
- `state` (String) 

## Import

```shell
terraform import adatage_load_balancer_forwarding_rule.example ID
```
