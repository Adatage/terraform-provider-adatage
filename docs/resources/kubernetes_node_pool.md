---
page_title: "adatage_kubernetes_node_pool Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A pool of identical nodes of a cluster.
---

# adatage_kubernetes_node_pool (Resource)

A pool of identical nodes of a cluster. Import with cluster_id/node_pool_id.

## Example Usage

```terraform
resource "adatage_kubernetes_node_pool" "gpu" {
  cluster_id  = adatage_kubernetes_cluster.main.id
  name        = "gpu"
  series_code = "A"
  vcpus       = 8
  memory_mib  = 32768
  size        = 1
  min_size    = 1
  max_size    = 5

  labels = { team = "ml" }
  taints = [{ key = "dedicated", value = "ml", effect = "NoSchedule" }]
}
```

## Schema

### Required

- `cluster_id` (String) (changing it replaces the resource)
- `memory_mib` (Number) (changing it replaces the resource)
- `name` (String) (changing it replaces the resource)
- `series_code` (String) (changing it replaces the resource)
- `size` (Number) The number of nodes.
- `vcpus` (Number) (changing it replaces the resource)

### Optional

- `disk_gib` (Number) (changing it replaces the resource)
- `labels` (Map of String) 
- `max_pods` (Number) From 8 to 110, default 30. (changing it replaces the resource)
- `max_size` (Number) 
- `min_size` (Number) With max_size turns on the autoscaler.
- `taints` (Attributes List) 
  - `key` (String) 
  - `value` (String) 
  - `effect` (String) NoSchedule, PreferNoSchedule or NoExecute.

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) The id.
- `ready_nodes` (Number) 

## Import

```shell
terraform import adatage_kubernetes_node_pool.example CLUSTER_ID/ID
```
