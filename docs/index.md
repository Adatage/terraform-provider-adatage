---
page_title: "Provider: Adatage"
description: |-
  Manage Adatage Cloud: virtual machines, networks, load balancers, Kubernetes and access.
---

# Adatage Provider

The Adatage provider manages resources of [Adatage Cloud](https://cloud.adatage.com): compute instances and disks, networks, subnets, firewall rules, public IPs, load balancers, Kubernetes clusters and node pools, and access (roles, service accounts and keys). It works with Terraform and OpenTofu.

## Authentication

The provider signs in as a **service account**, the same way Google Cloud's do:

1. In the console, open *IAM & Access > Service Accounts*, create an account with the role it needs, then *Create key* and download the credentials file (JSON). The private key is shown once.
2. Give the file to the provider: `credentials_file`, `credentials` (the content) or the environment variables `ADATAGE_CREDENTIALS_FILE` / `ADATAGE_CREDENTIALS`.

The provider signs a short assertion with the key, trades it for an access token (valid for an hour, renewed on its own) and sends it with every call. The service account holds one role in one project, so it can only change that project; `project` defaults to the project in the credentials.

## Example Usage

```terraform
terraform {
  required_providers {
    adatage = {
      source = "adatage/adatage"
    }
  }
}

provider "adatage" {
  credentials_file = "adatage-credentials.json"
}

resource "adatage_network" "main" {
  name = "main"
}
```

## Schema

### Optional

- `access_token` (String, Sensitive) An access token, instead of credentials. Environment variable `ADATAGE_ACCESS_TOKEN`.
- `api_url` (String) The address of the API. Defaults to `https://api.cloud.adatage.com`; environment variable `ADATAGE_API_URL`.
- `credentials` (String, Sensitive) The credentials file of a service account (its JSON content, or the path of the file). Environment variable `ADATAGE_CREDENTIALS`.
- `credentials_file` (String) The path of the credentials file. Environment variable `ADATAGE_CREDENTIALS_FILE`.
- `project` (String) The id of the project to work in. Defaults to the project of the credentials; environment variable `ADATAGE_PROJECT`.

## Notes

- Creating instances, disks, networks and clusters waits until they are ready; deleting waits until they are gone.
- Attributes that only matter at creation (the `disks` and `nics` of an instance, the first `node_pool` of a cluster) are not read back from the API: drift in them is not detected.
- Ids are strings, as 64-bit numbers do not fit every tool.
