---
page_title: "adatage_snapshot Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A snapshot of an instance.
---

# adatage_snapshot (Resource)

A snapshot of an instance.

## Example Usage

```terraform
resource "adatage_snapshot" "before_upgrade" {
  snapshot_id = "before-upgrade"
  instance_id = adatage_instance.web.id
}
```

## Schema

### Required

- `instance_id` (String) The instance it is made from. (changing it replaces the resource; not read back from the API)
- `snapshot_id` (String) The short name: lowercase letters, numbers and hyphens. (changing it replaces the resource; not read back from the API)

### Optional

- `description` (String) (changing it replaces the resource)
- `display_name` (String) (changing it replaces the resource)

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The full name.
- `memory_mib` (Number) 
- `region_code` (String) 
- `series_code` (String) 
- `status` (String) creating, ready, deleting or error.
- `vcpus` (Number) 
- `zone_code` (String) 

## Import

```shell
terraform import adatage_snapshot.example ID
```
