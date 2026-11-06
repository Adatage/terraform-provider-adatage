---
page_title: "adatage_network Data Source - terraform-provider-adatage"
subcategory: ""
description: |-
  Finds a network by name or id.
---

# adatage_network (Data Source)

Finds a network by name or id.

## Example Usage

```terraform
data "adatage_network" "main" {
  name = "main"
}
```

## Schema

### Optional

- `id` (String) 
- `name` (String) 

### Read-Only

- `ipv6_enabled` (Boolean) 
- `mtu` (Number) 
- `state` (String) 

