---
page_title: "adatage_database_engine_versions Data Source - terraform-provider-adatage"
subcategory: ""
description: |-
  The versions the platform offers of a database engine.
---

# adatage_database_engine_versions (Data Source)

The versions the platform offers of a database engine.

## Example Usage

```terraform
data "adatage_database_engine_versions" "valkey" {
  engine = "valkey"
}
```

## Schema

### Required

- `engine` (String) postgres or valkey.

### Read-Only

- `default_version` (String) 
- `id` (String) 
- `versions` (List of String) 

