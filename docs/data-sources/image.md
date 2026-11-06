---
page_title: "adatage_image Data Source - terraform-provider-adatage"
subcategory: ""
description: |-
  Finds an image by name, or the newest public image of an operating system and version.
---

# adatage_image (Data Source)

Finds an image by name, or the newest public image of an operating system and version.

## Example Usage

```terraform
data "adatage_image" "debian" {
  os         = "debian"
  os_version = "13"
}
```

## Schema

### Optional

- `architecture` (String) 
- `name` (String) The name disks refer to.
- `os` (String) 
- `os_version` (String) 

### Read-Only

- `display_name` (String) 
- `id` (String) 
- `is_public` (Boolean) 
- `min_size_bytes` (Number) 

