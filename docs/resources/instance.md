---
page_title: "adatage_instance Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A virtual machine.
---

# adatage_instance (Resource)

A virtual machine. The disks and nics blocks are used when the instance is created and cannot be changed afterwards (changing them replaces it); name, vcpus, memory_mib and start can change in place.

## Example Usage

```terraform
data "adatage_image" "debian" {
  os         = "debian"
  os_version = "13"
}

resource "adatage_instance" "web" {
  name        = "web"
  region_code = "eu-central-01"
  zone_code   = "a"
  series_code = "general"
  vcpus       = 2
  memory_mib  = 4096

  ssh_key_names = [adatage_ssh_key.admin.name]

  disks = [{
    is_boot     = true
    auto_delete = true
    name        = "web-boot"
    size_gib    = 20
    image       = data.adatage_image.debian.name
  }]

  nics = [{
    subnet_id = adatage_subnet.app.id
  }]
}
```

## Schema

### Required

- `disks` (Attributes List) The disks: a new disk (size_gib, image), or an existing one (existing_disk). (changing it replaces the resource; not read back from the API)
  - `device_index` (Number) 
  - `is_boot` (Boolean) 
  - `auto_delete` (Boolean) 
  - `kind` (String) disk (default) or cdrom.
  - `name` (String) The name of a new disk.
  - `size_gib` (Number) 
  - `storage_pool` (String) 
  - `image` (String) A public image to clone.
  - `existing_disk` (String) zones/ZONE/disks/DISK of a disk to attach.
- `memory_mib` (Number) 
- `name` (String) 
- `nics` (Attributes List) The network interfaces. (changing it replaces the resource; not read back from the API)
  - `device_index` (Number) 
  - `network_id` (String) 
  - `subnet_id` (String) 
  - `ip_type` (String) ipv4_only, ipv6_only or dual_stack.
  - `ipv4_address` (String) 
  - `ipv6_address` (String) 
  - `ipv4_aliases` (List of String) 
- `region_code` (String) The region, for example eu-central-01. (changing it replaces the resource)
- `series_code` (String) The machine series. (changing it replaces the resource)
- `vcpus` (Number) 
- `zone_code` (String) The zone letter, for example a. (changing it replaces the resource)

### Optional

- `hostname` (String) (changing it replaces the resource)
- `max_memory_mib` (Number) (changing it replaces the resource)
- `max_vcpus` (Number) The ceiling for adding vCPUs while running. (changing it replaces the resource)
- `restart_on_failure` (Boolean) (changing it replaces the resource)
- `secure_boot` (Boolean) (changing it replaces the resource)
- `ssh_key_names` (List of String) Names of adatage_ssh_key resources the guest authorizes. (changing it replaces the resource; not read back from the API)
- `start` (Boolean) Whether the instance runs (default true). Changing it starts or stops it. (not read back from the API)

### Read-Only

- `created_at` (String) When it was created.
- `etag` (String) 
- `id` (String) The id.
- `primary_ipv4` (String) The private IPv4 address of the first interface.
- `state` (String) 

## Import

```shell
terraform import adatage_instance.example ID
```
