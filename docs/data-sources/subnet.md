---
page_title: "adatage_subnet Data Source - terraform-provider-adatage"
subcategory: ""
description: |-
  Finds a subnet by name or id.
---

# adatage_subnet (Data Source)

Finds a subnet by name or id.

## Example Usage

```terraform
data "adatage_subnet" "app" {
  name = "app"
}
```

## Schema

### Optional

- `id` (String) 
- `name` (String) 
- `network_id` (String) 

### Read-Only

- `ip_type` (String) 
- `ipv4_cidr` (String) 
- `ipv6_cidr` (String) 

