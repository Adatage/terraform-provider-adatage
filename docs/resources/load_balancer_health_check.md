---
page_title: "adatage_load_balancer_health_check Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A health check that a backend service uses to decide which backends receive traffic.
---

# adatage_load_balancer_health_check (Resource)

A health check that a backend service uses to decide which backends receive traffic.

## Example Usage

```terraform
resource "adatage_load_balancer_health_check" "http" {
  name         = "http"
  protocol     = "http"
  port         = 80
  request_path = "/healthz"
}
```

## Schema

### Required

- `name` (String) (changing it replaces the resource)
- `port` (Number) 
- `protocol` (String) tcp or http.

### Optional

- `healthy_threshold` (Number) 1 to 10, default 2.
- `interval_seconds` (Number) 2 to 300, default 5.
- `request_path` (String) http only; / by default.
- `timeout_seconds` (Number) 1 to the interval, default 3.
- `unhealthy_threshold` (Number) 1 to 10, default 3.

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.

## Import

```shell
terraform import adatage_load_balancer_health_check.example ID
```
