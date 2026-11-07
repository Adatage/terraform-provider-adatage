---
page_title: "adatage_database_backup Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A manual backup of a database instance, kept until its expiry (the instance's retention).
---

# adatage_database_backup (Resource)

A manual backup of a database instance, kept until its expiry (the instance's retention). Import with instance_id/backup_id.

## Example Usage

```terraform
resource "adatage_database_backup" "before_upgrade" {
  instance_id = adatage_database_instance.main.id
  description = "before the upgrade"
}
```

## Schema

### Required

- `instance_id` (String) (changing it replaces the resource)

### Optional

- `description` (String) (changing it replaces the resource)
- `wait_for_ready` (Boolean) Wait until the backup is ready (default true). (not read back from the API)

### Read-Only

- `created_at` (String) When it was created.
- `expire_time` (String) 
- `id` (String) The id.
- `kind` (String) 
- `message` (String) 
- `size_bytes` (Number) 
- `state` (String) running, ready or failed.

## Import

```shell
terraform import adatage_database_backup.example INSTANCE_ID/ID
```
