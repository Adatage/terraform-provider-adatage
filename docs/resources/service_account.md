---
page_title: "adatage_service_account Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A service account: an identity for programs that holds one role in the project.
---

# adatage_service_account (Resource)

A service account: an identity for programs that holds one role in the project.

## Example Usage

```terraform
resource "adatage_service_account" "ci" {
  name = "ci"
  role = "admin"
}
```

## Schema

### Required

- `name` (String) Lowercase letters, digits and hyphens. (changing it replaces the resource)
- `role` (String) The id of the role it holds in the project.

### Optional

- `description` (String) 
- `disabled` (Boolean) 
- `display_name` (String) 

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.

## Import

```shell
terraform import adatage_service_account.example ID
```
