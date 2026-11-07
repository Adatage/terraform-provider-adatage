---
page_title: "adatage_disk Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A block disk in a zone.
---

# adatage_disk (Resource)

A block disk in a zone. Attach it to an instance with the disks block of adatage_instance.

## Example Usage

```terraform
resource "adatage_disk" "data" {
  zone     = "eu-central-01a"
  disk_id   = "data"
  disk_type = "volumes-standard"
  size_gib  = 100
}
```

## Schema

### Required

- `disk_id` (String) The name of the disk in the zone: lowercase letters, numbers and hyphens. (changing it replaces the resource; not read back from the API)
- `disk_type` (String) The storage pool: volumes-standard or volumes-performance (see the pools of the region). (changing it replaces the resource)
- `size_gib` (Number) The size in GiB; a disk can only grow.
- `zone` (String) The zone, for example eu-central-01a. (changing it replaces the resource; not read back from the API)

### Optional

- `display_name` (String) 
- `image` (String) A public image to clone; empty for a blank disk. (changing it replaces the resource)

### Read-Only

- `created_at` (String) When it was created.
- `etag` (String) 
- `id` (String) zones/ZONE/disks/DISK.
- `phase` (String) 

## Import

```shell
terraform import adatage_disk.example zones/ZONE/disks/DISK
```
