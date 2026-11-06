---
page_title: "adatage_kubernetes_cluster Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A managed Kubernetes cluster with a highly available control plane.
---

# adatage_kubernetes_cluster (Resource)

A managed Kubernetes cluster with a highly available control plane. node_pool describes the first pool at creation; manage more pools (and later changes to the first one) with adatage_kubernetes_node_pool.

## Example Usage

```terraform
resource "adatage_kubernetes_cluster" "main" {
  name        = "main"
  region_code = "eu-central-01"
  zone_code   = "a"
  subnet_id   = adatage_subnet.app.id

  node_pool = {
    name        = "default"
    series_code = "general"
    vcpus       = 2
    memory_mib  = 4096
    size        = 3
  }
}

data "adatage_kubernetes_cluster_credentials" "main" {
  cluster_id = adatage_kubernetes_cluster.main.id
}
```

## Schema

### Required

- `name` (String) (changing it replaces the resource)
- `node_pool` (Attributes) The first node pool; only used when the cluster is created. (not read back from the API)
  - `name` (String) 
  - `series_code` (String) 
  - `vcpus` (Number) 
  - `memory_mib` (Number) 
  - `disk_gib` (Number) 
  - `size` (Number) 
  - `max_pods` (Number) 
  - `labels` (Map of String) 
- `region_code` (String) (changing it replaces the resource)
- `subnet_id` (String) The subnet of the nodes; pods get addresses from it. (changing it replaces the resource)
- `zone_code` (String) (changing it replaces the resource)

### Optional

- `control_plane_count` (Number) 1 or 3. (changing it replaces the resource)
- `kubernetes_version` (String) Setting a newer version upgrades the cluster.

### Read-Only

- `created_at` (String) When it was created.
- `endpoint` (String) The address of the Kubernetes API.
- `id` (String) The id.
- `message` (String) 
- `node_count` (Number) 
- `ready_node_count` (Number) 
- `state` (String) provisioning, running, error or deleting.

## Import

```shell
terraform import adatage_kubernetes_cluster.example ID
```
