---
page_title: "adatage_iam_role Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A custom role: a named set of permissions that can be given to members and service accounts.
---

# adatage_iam_role (Resource)

A custom role: a named set of permissions that can be given to members and service accounts.

## Example Usage

```terraform
resource "adatage_iam_role" "viewer" {
  name        = "Viewer"
  permissions = ["compute.instances.list", "compute.instances.get", "network.networks.list"]
}
```

## Schema

### Required

- `name` (String) The display name; the id is made from it.
- `permissions` (List of String) Permission names such as compute.instances.list.

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.

## Import

```shell
terraform import adatage_iam_role.example ID
```
