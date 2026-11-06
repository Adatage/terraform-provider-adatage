---
page_title: "adatage_ssh_key Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A public SSH key of the project that instances can be given.
---

# adatage_ssh_key (Resource)

A public SSH key of the project that instances can be given.

## Example Usage

```terraform
resource "adatage_ssh_key" "admin" {
  name       = "admin"
  public_key = file("~/.ssh/id_ed25519.pub")
}
```

## Schema

### Required

- `name` (String) The name of the key. (changing it replaces the resource)
- `public_key` (String) The public key in the authorized_keys format. (changing it replaces the resource)

### Read-Only

- `comment` (String) 
- `created_at` (String) When it was created.
- `fingerprint` (String) SHA256:... as ssh-keygen prints it.
- `id` (String) The name of the key.
- `key_type` (String) ssh-ed25519, ssh-rsa and so on.

## Import

```shell
terraform import adatage_ssh_key.example ID
```
