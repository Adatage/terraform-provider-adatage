---
page_title: "adatage_load_balancer_backend_service Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  The instances a forwarding rule sends traffic to, with a health check.
---

# adatage_load_balancer_backend_service (Resource)

The instances a forwarding rule sends traffic to, with a health check.

## Example Usage

```terraform
resource "adatage_load_balancer_backend_service" "web" {
  name            = "web"
  health_check_id = adatage_load_balancer_health_check.http.id
  affinity        = "client_ip"

  backends = [{ instance_id = adatage_instance.web.id }]
}
```

## Schema

### Required

- `name` (String) (changing it replaces the resource)

### Optional

- `affinity` (String) none or client_ip.
- `backends` (Attributes List) The instances (and interfaces) that get traffic.
  - `instance_id` (String) 
  - `nic_id` (String) 
  - `weight` (Number) 
- `health_check_id` (String) 

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.

## Import

```shell
terraform import adatage_load_balancer_backend_service.example ID
```
