---
page_title: "adatage_database_instance Resource - terraform-provider-adatage"
subcategory: ""
description: |-
  A managed database (PostgreSQL or Valkey) on virtual machines the platform runs in your subnet: one primary and up to two replicas with automatic failover, a private address that stays the same, daily backups and point-in-time restore for PostgreSQL.
---

# adatage_database_instance (Resource)

A managed database (PostgreSQL or Valkey) on virtual machines the platform runs in your subnet: one primary and up to two replicas with automatic failover, a private address that stays the same, daily backups and point-in-time restore for PostgreSQL.

## Example Usage

```terraform
data "adatage_database_engine_versions" "postgres" {
  engine = "postgres"
}

resource "adatage_database_instance" "main" {
  name        = "main"
  engine      = "postgres"
  version     = data.adatage_database_engine_versions.postgres.default_version
  region_code = "eu-central-01"
  zone_code   = "a"
  subnet_id   = adatage_subnet.app.id

  vcpus      = 2
  memory_mib = 4096
  disk_gib   = 50
  node_count = 2 # a primary and a replica, with automatic failover

  backup_window         = "03:00"
  backup_retention_days = 14
}

output "database_uri" {
  value     = adatage_database_instance.main.uri
  sensitive = true
}
```

## Schema

### Required

- `disk_gib` (Number) Disk per node; it can only grow.
- `engine` (String) postgres or valkey. (changing it replaces the resource)
- `memory_mib` (Number) Memory per node.
- `name` (String) Lowercase letters, numbers and hyphens.
- `region_code` (String) (changing it replaces the resource)
- `subnet_id` (String) The subnet that holds the nodes and the address. (changing it replaces the resource)
- `vcpus` (Number) vCPUs per node.
- `zone_code` (String) (changing it replaces the resource)

### Optional

- `address` (String) The private IPv4 address of the database in the subnet; the next free one when empty. It stays through failovers. (changing it replaces the resource)
- `allowed_cidrs` (List of String) IPv4 ranges the public address accepts connections from; required with public_access.
- `backup_disabled` (Boolean) 
- `backup_point_in_time` (Boolean) Continuous log archiving for point-in-time restore (PostgreSQL).
- `backup_retention_days` (Number) How long backups are kept, 1 to 35, 7 by default.
- `backup_window` (String) Daily backup time (UTC, HH:MM), 03:00 by default.
- `durable` (Boolean) Valkey: keep the data (needed for replicas); otherwise it is a cache on one node. (changing it replaces the resource; not read back from the API)
- `keep_backups` (Boolean) Keep the backups of the instance for their retention after it is deleted. (not read back from the API)
- `labels` (Map of String) 
- `node_count` (Number) 1 to 3: a primary and its replicas.
- `public_access` (Boolean) Whether the internet may connect (to a public address translated to the private one).
- `version` (String) The engine version (see the data source adatage_database_engine_versions); the default when empty. (changing it replaces the resource)
- `wait_for_running` (Boolean) Wait until the database is running after a change (default true). (not read back from the API)

### Read-Only

- `created_at` (String) When it was created.
- `host` (String) 
- `id` (String) The id.
- `message` (String) 
- `network_id` (String) 
- `password` (String) The administrator's password (needs the permission database.instances.credentials). (sensitive)
- `port` (Number) 
- `public_address` (String) 
- `state` (String) creating, running, updating, failing_over, restoring, deleting or error.
- `uri` (String) The connection URI, with the password in it. (sensitive)
- `username` (String) 

## Import

```shell
terraform import adatage_database_instance.example ID
```
