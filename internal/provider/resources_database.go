package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
)

const svcDatabase = "cloud.external.database.v1.DatabaseService"

var databaseRunning = states{path: "state", ready: []string{"STATE_RUNNING"}, failed: []string{"STATE_ERROR"}}

// databaseConnection adds the connection details (host, user, password, URI) to the instance's JSON, as far as
// the caller may read them and the instance has an address; without them the attributes are empty.
func databaseConnection(ctx context.Context, c *client.Client, id string, obj map[string]any) {
	if str(obj, "address") == "" || str(obj, "state") == "STATE_DELETING" {
		return
	}
	if info, err := c.Call(ctx, svcDatabase, "GetConnectionInfo", map[string]any{"instanceId": id}); err == nil {
		obj["connection"] = info
	}
}

func databaseResource() *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcDatabase, "GetInstance", map[string]any{"instanceId": id})
		if err != nil {
			return nil, gone(err)
		}
		databaseConnection(ctx, c, id, obj)
		return obj, nil
	}
	wait := func(ctx context.Context, c *client.Client, id string, p Values) error {
		if p.Has("wait_for_running") && !p.Bool("wait_for_running") {
			return nil
		}
		time.Sleep(2 * time.Second)
		return waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, databaseRunning)
	}
	backup := func(p Values) map[string]any {
		return map[string]any{"windowStart": p.Str("backup_window"), "retentionDays": p.Int("backup_retention_days"), "pointInTime": p.Bool("backup_point_in_time"), "disabled": p.Bool("backup_disabled")}
	}
	return &Resource{
		Name: "database_instance",
		Desc: "A managed database (PostgreSQL or Valkey) on virtual machines the platform runs in your subnet: one primary and up to two replicas with automatic failover, a private address that stays the same, daily backups and point-in-time restore for PostgreSQL.",
		Attrs: []Attr{
			idAttr("instanceId"),
			{Name: "name", Kind: String, Required: true, Desc: "Lowercase letters, numbers and hyphens."},
			{Name: "engine", Kind: String, Required: true, ForceNew: true, Enum: "ENGINE_", Desc: "postgres or valkey."},
			{Name: "version", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true, Desc: "The engine version (see the data source adatage_database_engine_versions); the default when empty."},
			{Name: "region_code", Kind: String, Required: true, ForceNew: true},
			{Name: "zone_code", Kind: String, Required: true, ForceNew: true},
			{Name: "subnet_id", Kind: String, Required: true, ForceNew: true, Desc: "The subnet that holds the nodes and the address."},
			{Name: "address", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true, KeepPrior: true, Desc: "The private IPv4 address of the database in the subnet; the next free one when empty. It stays through failovers."},
			{Name: "vcpus", Kind: Int, Required: true, Desc: "vCPUs per node."},
			{Name: "memory_mib", Kind: Int, Required: true, Desc: "Memory per node."},
			{Name: "disk_gib", Kind: Int, Required: true, Desc: "Disk per node; it can only grow."},
			{Name: "node_count", Kind: Int, Optional: true, Computed: true, Desc: "1 to 3: a primary and its replicas."},
			{Name: "durable", Kind: Bool, Optional: true, ForceNew: true, Local: true, Desc: "Valkey: keep the data (needed for replicas); otherwise it is a cache on one node."},
			{Name: "public_access", Kind: Bool, Optional: true, Computed: true, Desc: "Whether the internet may connect (to a public address translated to the private one)."},
			{Name: "allowed_cidrs", Kind: StringList, Optional: true, Computed: true, Desc: "IPv4 ranges the public address accepts connections from; required with public_access."},
			{Name: "backup_window", Kind: String, Optional: true, Computed: true, Path: "backup.windowStart", Desc: "Daily backup time (UTC, HH:MM), 03:00 by default."},
			{Name: "backup_retention_days", Kind: Int, Optional: true, Computed: true, Path: "backup.retentionDays", Desc: "How long backups are kept, 1 to 35, 7 by default."},
			{Name: "backup_point_in_time", Kind: Bool, Optional: true, Computed: true, Path: "backup.pointInTime", Desc: "Continuous log archiving for point-in-time restore (PostgreSQL)."},
			{Name: "backup_disabled", Kind: Bool, Optional: true, Computed: true, Path: "backup.disabled"},
			{Name: "labels", Kind: StringMap, Optional: true, Computed: true},
			{Name: "wait_for_running", Kind: Bool, Optional: true, Local: true, Desc: "Wait until the database is running after a change (default true)."},
			{Name: "keep_backups", Kind: Bool, Optional: true, Local: true, Desc: "Keep the backups of the instance for their retention after it is deleted."},
			{Name: "port", Kind: Int, Computed: true, Stable: true},
			{Name: "public_address", Kind: String, Computed: true},
			{Name: "network_id", Kind: String, Computed: true, Stable: true},
			{Name: "state", Kind: String, Computed: true, Enum: "STATE_", Desc: "creating, running, updating, failing_over, restoring, deleting or error."},
			{Name: "message", Kind: String, Computed: true},
			{Name: "host", Kind: String, Computed: true, Path: "connection.host"},
			{Name: "username", Kind: String, Computed: true, Path: "connection.username"},
			{Name: "password", Kind: String, Computed: true, Sensitive: true, Path: "connection.password", Desc: "The administrator's password (needs the permission database.instances.credentials)."},
			{Name: "uri", Kind: String, Computed: true, Sensitive: true, Path: "connection.uri", Desc: "The connection URI, with the password in it."},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{
				"name": p.Str("name"), "engine": Enum("ENGINE_", p.Str("engine")), "regionCode": p.Str("region_code"), "zoneCode": p.Str("zone_code"), "subnetId": p.Str("subnet_id"),
				"vcpus": p.Int("vcpus"), "memoryMib": p.Int("memory_mib"), "diskGib": p.Int("disk_gib"),
			}
			set(body, "version", p.Str("version"))
			set(body, "address", p.Str("address"))
			set(body, "nodeCount", p.Int("node_count"))
			if p.Has("durable") {
				body["durable"] = p.Bool("durable")
			}
			if p.Has("public_access") {
				body["publicAccess"] = p.Bool("public_access")
			}
			if l := p.Strings("allowed_cidrs"); len(l) > 0 {
				body["allowedCidrs"] = strList(l)
			}
			if p.Has("backup_window") || p.Has("backup_retention_days") || p.Has("backup_point_in_time") || p.Has("backup_disabled") {
				b := backup(p)
				if !p.Has("backup_point_in_time") {
					b["pointInTime"] = p.Str("engine") == "postgres"
				}
				body["backup"] = b
			}
			if m := strMap(p.Map("labels")); len(m) > 0 {
				body["labels"] = m
			}
			obj, err := c.Call(ctx, svcDatabase, "CreateInstance", body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "instanceId")
			return ids("id", id), wait(ctx, c, id, p)
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			m := merge(s, p)
			var paths []string
			body := map[string]any{"instanceId": s.Str("id")}
			if p.Has("name") && p.Str("name") != s.Str("name") {
				paths, body["name"] = append(paths, "name"), p.Str("name")
			}
			if p.Int("vcpus") != s.Int("vcpus") {
				paths, body["vcpus"] = append(paths, "vcpus"), p.Int("vcpus")
			}
			if p.Int("memory_mib") != s.Int("memory_mib") {
				paths, body["memoryMib"] = append(paths, "memory_mib"), p.Int("memory_mib")
			}
			if p.Int("disk_gib") != s.Int("disk_gib") {
				paths, body["diskGib"] = append(paths, "disk_gib"), p.Int("disk_gib")
			}
			if p.Has("node_count") && p.Int("node_count") != s.Int("node_count") {
				paths, body["nodeCount"] = append(paths, "node_count"), p.Int("node_count")
			}
			if (p.Has("public_access") && p.Bool("public_access") != s.Bool("public_access")) || strings.Join(m.Strings("allowed_cidrs"), ",") != strings.Join(s.Strings("allowed_cidrs"), ",") {
				paths = append(paths, "public_access", "allowed_cidrs")
				body["publicAccess"] = m.Bool("public_access")
				body["allowedCidrs"] = strList(m.Strings("allowed_cidrs"))
			}
			if m.Str("backup_window") != s.Str("backup_window") || m.Int("backup_retention_days") != s.Int("backup_retention_days") || m.Bool("backup_point_in_time") != s.Bool("backup_point_in_time") || m.Bool("backup_disabled") != s.Bool("backup_disabled") {
				paths, body["backup"] = append(paths, "backup"), backup(m)
			}
			if p.Has("labels") && fmt.Sprint(strMap(p.Map("labels"))) != fmt.Sprint(strMap(s.Map("labels"))) {
				paths, body["labels"] = append(paths, "labels"), strMap(p.Map("labels"))
			}
			if len(paths) == 0 {
				return nil
			}
			body["updateMask"] = fieldMask(paths...)
			if _, err := c.Call(ctx, svcDatabase, "UpdateInstance", body); err != nil {
				return err
			}
			return wait(ctx, c, s.Str("id"), m)
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, svcDatabase, "DeleteInstance", map[string]any{"instanceId": s.Str("id"), "keepBackups": s.Bool("keep_backups")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

func databaseBackupResource() *Resource {
	read := func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
		list, err := c.Call(ctx, svcDatabase, "ListBackups", map[string]any{"instanceId": s.Str("instance_id")})
		if err != nil {
			return nil, gone(err)
		}
		return listFind(list, "backups", "backupId", s.Str("id"))
	}
	return &Resource{
		Name:         "database_backup",
		Desc:         "A manual backup of a database instance, kept until its expiry (the instance's retention). Import with instance_id/backup_id.",
		ImportFields: []string{"instance_id", "id"},
		Attrs: []Attr{
			idAttr("backupId"),
			{Name: "instance_id", Kind: String, Required: true, ForceNew: true},
			{Name: "description", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true},
			{Name: "wait_for_ready", Kind: Bool, Optional: true, Local: true, Desc: "Wait until the backup is ready (default true)."},
			{Name: "state", Kind: String, Computed: true, Enum: "STATE_", Desc: "running, ready or failed."},
			{Name: "kind", Kind: String, Computed: true, Stable: true, Enum: "KIND_"},
			{Name: "size_bytes", Kind: Int, Computed: true},
			{Name: "message", Kind: String, Computed: true},
			{Name: "expire_time", Kind: String, Computed: true, Stable: true, Path: "expireTime"},
			createdAttr("createTime"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{"instanceId": p.Str("instance_id")}
			set(body, "description", p.Str("description"))
			obj, err := c.Call(ctx, svcDatabase, "CreateBackup", body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "backupId")
			if p.Has("wait_for_ready") && !p.Bool("wait_for_ready") {
				return ids("id", id), nil
			}
			return ids("id", id), waitState(ctx, func() (map[string]any, error) {
				return read(ctx, c, Values{"instance_id": p.Str("instance_id"), "id": id})
			}, states{path: "state", ready: []string{"STATE_READY"}, failed: []string{"STATE_FAILED"}})
		},
		Read: read,
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			_, err := c.Call(ctx, svcDatabase, "DeleteBackup", map[string]any{"instanceId": s.Str("instance_id"), "backupId": s.Str("id")})
			return gone(err)
		},
	}
}

func databaseEngineVersionsDataSource() *DataSource {
	return &DataSource{
		Name: "database_engine_versions",
		Desc: "The versions the platform offers of a database engine.",
		Attrs: []Attr{
			{Name: "id", Kind: String, Computed: true},
			{Name: "engine", Kind: String, Required: true, Desc: "postgres or valkey."},
			{Name: "versions", Kind: StringList, Computed: true},
			{Name: "default_version", Kind: String, Computed: true},
		},
		Read: func(ctx context.Context, c *client.Client, cfg Values) (map[string]any, error) {
			list, err := c.Call(ctx, svcDatabase, "ListEngineVersions", map[string]any{})
			if err != nil {
				return nil, err
			}
			want := Enum("ENGINE_", cfg.Str("engine"))
			versions, def := []any{}, ""
			for _, it := range asList(list["versions"]) {
				if m, ok := it.(map[string]any); ok && str(m, "engine") == want {
					versions = append(versions, str(m, "version"))
					if d, _ := m["isDefault"].(bool); d {
						def = str(m, "version")
					}
				}
			}
			if len(versions) == 0 {
				return nil, fmt.Errorf("the platform has no engine %q", cfg.Str("engine"))
			}
			return map[string]any{"id": cfg.Str("engine"), "versions": versions, "defaultVersion": def}, nil
		},
	}
}
