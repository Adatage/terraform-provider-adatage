---
page_title: "adatage_kubernetes_cluster_credentials Data Source - terraform-provider-adatage"
subcategory: ""
description: |-
  The kubeconfig and the CA certificate of a cluster, for the kubernetes and helm providers.
---

# adatage_kubernetes_cluster_credentials (Data Source)

The kubeconfig and the CA certificate of a cluster, for the kubernetes and helm providers. The kubeconfig signs in through adatagecli.

## Example Usage

```terraform
data "adatage_kubernetes_cluster_credentials" "main" {
  cluster_id = adatage_kubernetes_cluster.main.id
}

provider "kubernetes" {
  host                   = data.adatage_kubernetes_cluster_credentials.main.endpoint
  cluster_ca_certificate = data.adatage_kubernetes_cluster_credentials.main.ca_certificate

  exec {
    api_version = "client.authentication.k8s.io/v1"
    command     = "adatagecli"
    args        = ["kubernetes", "get-token", "--cluster", adatage_kubernetes_cluster.main.id]
  }
}
```

## Schema

### Required

- `cluster_id` (String) 

### Read-Only

- `ca_certificate` (String) 
- `endpoint` (String) 
- `id` (String) 
- `kubeconfig` (String) (sensitive)

