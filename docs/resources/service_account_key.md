---
page_title: "adatage_service_account_key Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A key of a service account.
---

# adatage_service_account_key (Resource)

A key of a service account. The credentials (the file to give to the provider, Terraform or a script) are only known when the key is created and are kept, sensitive, in the state.

## Example Usage

```terraform
resource "adatage_service_account_key" "ci" {
  service_account_id = adatage_service_account.ci.id
}

output "ci_credentials" {
  value     = adatage_service_account_key.ci.credentials
  sensitive = true
}
```

## Schema

### Required

- `service_account_id` (String) (changing it replaces the resource)

### Optional

- `description` (String) (changing it replaces the resource)
- `expire_time` (String) RFC 3339 time after which the key stops working; it does not expire when empty. (changing it replaces the resource; not read back from the API)

### Read-Only

- `created_at` (String) When it was created.
- `credentials` (String) The credentials file (JSON). (sensitive; not read back from the API)
- `id` (String) The id.

## Import

```shell
terraform import adatage_service_account_key.example SERVICE_ACCOUNT_ID/ID
```
