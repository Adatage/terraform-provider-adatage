---
page_title: "adatage_template Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A template of an instance (its shape, disks and interfaces) to make instances from.
---

# adatage_template (Resource)

A template of an instance (its shape, disks and interfaces) to make instances from.

## Example Usage

```terraform
resource "adatage_template" "web" {
  template_id  = "web"
  instance_id  = adatage_instance.web.id
  display_name = "Web server"
}
```

## Schema

### Required

- `instance_id` (String) The instance it is made from. (changing it replaces the resource; not read back from the API)
- `template_id` (String) The short name: lowercase letters, numbers and hyphens. (changing it replaces the resource; not read back from the API)

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
terraform import adatage_template.example ID
```
