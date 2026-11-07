package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
)

var instanceRunning = states{path: "status.state", ready: []string{"PHASE_STATE_RUNNING"}, failed: []string{"PHASE_STATE_INTERNAL_ERROR", "PHASE_STATE_GUEST_PANIC"}}
var instanceStopped = states{path: "status.state", ready: []string{"PHASE_STATE_STOPPED"}, failed: []string{"PHASE_STATE_INTERNAL_ERROR", "PHASE_STATE_GUEST_PANIC"}}

func diskResource() *Resource {
	get := func(ctx context.Context, c *client.Client, name string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcDisk, "GetDisk", map[string]any{"name": name})
		return obj, gone(err)
	}
	return &Resource{
		Name: "disk",
		Desc: "A block disk in a zone. Attach it to an instance with the disks block of adatage_instance.",
		ImportParse: func(id string) (Values, error) {
			parts := strings.Split(id, "/")
			switch {
			case len(parts) == 4 && parts[0] == "zones" && parts[2] == "disks":
				return Values{"id": id, "zone": parts[1], "disk_id": parts[3]}, nil
			case len(parts) == 2:
				return Values{"id": fmt.Sprintf("zones/%s/disks/%s", parts[0], parts[1]), "zone": parts[0], "disk_id": parts[1]}, nil
			}
			return nil, errors.New("expected zone/disk or zones/ZONE/disks/DISK")
		},
		Attrs: []Attr{
			{Name: "id", Kind: String, Computed: true, Stable: true, Path: "name", Desc: "zones/ZONE/disks/DISK."},
			{Name: "zone", Kind: String, Required: true, ForceNew: true, Local: true, Desc: "The zone, for example eu-central-01a."},
			{Name: "disk_id", Kind: String, Required: true, ForceNew: true, Local: true, Desc: "The name of the disk in the zone: lowercase letters, numbers and hyphens."},
			{Name: "display_name", Kind: String, Optional: true, Computed: true},
			{Name: "size_gib", Kind: Int, Required: true, Path: "spec.sizeGib", Desc: "The size in GiB; a disk can only grow."},
			{Name: "disk_type", Kind: String, Required: true, ForceNew: true, Path: "spec.diskType", Desc: "The storage pool: volumes-standard or volumes-performance (see the pools of the region)."},
			{Name: "image", Kind: String, Optional: true, ForceNew: true, Path: "spec.image", Desc: "A public image to clone; empty for a blank disk."},
			{Name: "phase", Kind: String, Computed: true, Path: "status.phase", Enum: "DISK_PHASE_"},
			{Name: "etag", Kind: String, Computed: true},
			createdAttr("createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			spec := map[string]any{"sizeGib": p.Int("size_gib"), "diskType": p.Str("disk_type")}
			set(spec, "image", p.Str("image"))
			disk := map[string]any{"spec": spec}
			set(disk, "displayName", p.Str("display_name"))
			op, err := c.Call(ctx, svcDisk, "CreateDisk", map[string]any{"zone": p.Str("zone"), "diskId": p.Str("disk_id"), "disk": disk})
			if err != nil {
				return nil, err
			}
			name := fmt.Sprintf("zones/%s/disks/%s", p.Str("zone"), p.Str("disk_id"))
			if _, err := c.WaitOperation(ctx, op); err != nil {
				return ids("id", name), err
			}
			return ids("id", name), nil
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			name := s.Str("id")
			etag := s.Str("etag")
			if p.Has("display_name") && p.Str("display_name") != s.Str("display_name") {
				op, err := c.Call(ctx, svcDisk, "UpdateDisk", map[string]any{"disk": map[string]any{"name": name, "displayName": p.Str("display_name"), "etag": etag}, "updateMask": fieldMask("display_name")})
				if err != nil {
					return err
				}
				if _, err := c.WaitOperation(ctx, op); err != nil {
					return err
				}
				obj, err := get(ctx, c, name)
				if err != nil {
					return err
				}
				etag = str(obj, "etag")
			}
			if p.Int("size_gib") != s.Int("size_gib") {
				op, err := c.Call(ctx, svcDisk, "ResizeDisk", map[string]any{"name": name, "etag": etag, "sizeGib": p.Int("size_gib")})
				if err != nil {
					return err
				}
				_, err = c.WaitOperation(ctx, op)
				return err
			}
			return nil
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			obj, err := get(ctx, c, s.Str("id"))
			if err != nil {
				return err
			}
			op, err := c.Call(ctx, svcDisk, "DeleteDisk", map[string]any{"name": s.Str("id"), "etag": str(obj, "etag")})
			if err != nil {
				return gone(err)
			}
			if _, err := c.WaitOperation(ctx, op); err != nil {
				return err
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

func instanceResource() *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, svcCompute, "GetInstance", map[string]any{"instanceId": id})
		return obj, gone(err)
	}
	return &Resource{
		Name: "instance",
		Desc: "A virtual machine. The disks and nics blocks are used when the instance is created and cannot be changed afterwards (changing them replaces it); name, vcpus, memory_mib and start can change in place.",
		Attrs: []Attr{
			idAttr("instanceId"),
			{Name: "name", Kind: String, Required: true, Path: "metadata.name"},
			{Name: "region_code", Kind: String, Required: true, ForceNew: true, Path: "metadata.regionCode", Desc: "The region, for example eu-central-01."},
			{Name: "zone_code", Kind: String, Required: true, ForceNew: true, Path: "metadata.zoneCode", Desc: "The zone letter, for example a."},
			{Name: "series_code", Kind: String, Required: true, ForceNew: true, Path: "metadata.seriesCode", Desc: "The machine series."},
			{Name: "vcpus", Kind: Int, Required: true, Path: "spec.vcpus"},
			{Name: "memory_mib", Kind: Int, Required: true, Path: "spec.memoryMib"},
			{Name: "max_vcpus", Kind: Int, Optional: true, Computed: true, Stable: true, ForceNew: true, Path: "spec.maxVcpus", Desc: "The ceiling for adding vCPUs while running."},
			{Name: "max_memory_mib", Kind: Int, Optional: true, Computed: true, Stable: true, ForceNew: true, Path: "spec.maxMemoryMib"},
			{Name: "secure_boot", Kind: Bool, Optional: true, Computed: true, Stable: true, ForceNew: true, Path: "spec.secureBoot"},
			{Name: "restart_on_failure", Kind: Bool, Optional: true, Computed: true, Stable: true, ForceNew: true, Path: "spec.restartOnFailure"},
			{Name: "start", Kind: Bool, Optional: true, Local: true, Desc: "Whether the instance runs (default true). Changing it starts or stops it."},
			{Name: "hostname", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true, Path: "spec.initialization.hostname"},
			{Name: "ssh_key_names", Kind: StringList, Optional: true, ForceNew: true, Local: true, Desc: "Names of adatage_ssh_key resources the guest authorizes."},
			{Name: "disks", Kind: List, Required: true, ForceNew: true, Local: true, Desc: "The disks: a new disk (size_gib, image), or an existing one (existing_disk).", Nested: []Attr{
				{Name: "device_index", Kind: Int, Optional: true},
				{Name: "is_boot", Kind: Bool, Optional: true},
				{Name: "auto_delete", Kind: Bool, Optional: true},
				{Name: "kind", Kind: String, Optional: true, Desc: "disk (default) or cdrom."},
				{Name: "name", Kind: String, Optional: true, Desc: "The name of a new disk."},
				{Name: "size_gib", Kind: Int, Optional: true},
				{Name: "storage_pool", Kind: String, Optional: true, Desc: "The storage pool of a new disk (required for one): volumes-standard or volumes-performance."},
				{Name: "image", Kind: String, Optional: true, Desc: "A public image to clone."},
				{Name: "existing_disk", Kind: String, Optional: true, Desc: "zones/ZONE/disks/DISK of a disk to attach."},
			}},
			{Name: "nics", Kind: List, Required: true, ForceNew: true, Local: true, Desc: "The network interfaces.", Nested: []Attr{
				{Name: "device_index", Kind: Int, Optional: true},
				{Name: "network_id", Kind: String, Optional: true},
				{Name: "subnet_id", Kind: String, Required: true},
				{Name: "ip_type", Kind: String, Optional: true, Desc: "ipv4_only, ipv6_only or dual_stack."},
				{Name: "ipv4_address", Kind: String, Optional: true},
				{Name: "ipv6_address", Kind: String, Optional: true},
				{Name: "ipv4_aliases", Kind: StringList, Optional: true},
			}},
			{Name: "primary_ipv4", Kind: String, Computed: true, Path: "nics.0.status.ipv4Address", Desc: "The private IPv4 address of the first interface."},
			{Name: "state", Kind: String, Computed: true, Path: "status.state", Enum: "PHASE_STATE_"},
			{Name: "etag", Kind: String, Computed: true},
			createdAttr("metadata.createdAt"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			start := !p.Has("start") || p.Bool("start")
			body := map[string]any{
				"start": start, "name": p.Str("name"), "vcpus": p.Int("vcpus"), "memoryMib": p.Int("memory_mib"),
				"regionCode": p.Str("region_code"), "zoneCode": p.Str("zone_code"), "seriesCode": p.Str("series_code"),
			}
			set(body, "maxVcpus", p.Int("max_vcpus"))
			set(body, "maxMemoryMib", p.Int("max_memory_mib"))
			if p.Has("secure_boot") {
				body["secureBoot"] = p.Bool("secure_boot")
			}
			if p.Has("restart_on_failure") {
				body["restartOnFailure"] = p.Bool("restart_on_failure")
			}
			var disks []any
			for i, d := range p.Objs("disks") {
				disk := map[string]any{"deviceIndex": d.Int("device_index"), "isBoot": d.Bool("is_boot"), "autoDelete": d.Bool("auto_delete")}
				if d.Has("device_index") == false {
					disk["deviceIndex"] = int64(i)
				}
				set(disk, "kind", d.Str("kind"))
				if d.Str("existing_disk") != "" {
					disk["existingDisk"] = d.Str("existing_disk")
				} else {
					nd := map[string]any{"sizeGib": d.Int("size_gib")}
					set(nd, "name", d.Str("name"))
					set(nd, "storagePool", d.Str("storage_pool"))
					set(nd, "image", d.Str("image"))
					disk["newDisk"] = nd
				}
				disks = append(disks, disk)
			}
			body["disks"] = disks
			var nics []any
			for i, n := range p.Objs("nics") {
				nic := map[string]any{"subnetId": n.Str("subnet_id"), "deviceIndex": n.Int("device_index")}
				if !n.Has("device_index") {
					nic["deviceIndex"] = int64(i)
				}
				set(nic, "networkId", n.Str("network_id"))
				set(nic, "ipType", Enum("IP_TYPE_", n.Str("ip_type")))
				set(nic, "ipv4Address", n.Str("ipv4_address"))
				set(nic, "ipv6Address", n.Str("ipv6_address"))
				if a := n.Strings("ipv4_aliases"); len(a) > 0 {
					nic["ipv4Aliases"] = strList(a)
				}
				nics = append(nics, nic)
			}
			body["nics"] = nics
			init := map[string]any{}
			set(init, "hostname", p.Str("hostname"))
			var keys []any
			for _, k := range p.Strings("ssh_key_names") {
				keys = append(keys, map[string]any{"name": k})
			}
			if len(keys) > 0 {
				init["sshKeys"] = keys
			}
			if len(init) > 0 {
				body["initialization"] = init
			}
			op, err := c.Call(ctx, svcCompute, "CreateInstance", body)
			if err != nil {
				return nil, err
			}
			op, err = c.WaitOperation(ctx, op)
			id := lastSegment(str(op, "target"))
			if err != nil || id == "" {
				if err == nil {
					err = fmt.Errorf("the operation named no instance (%s)", str(op, "target"))
				}
				if id != "" {
					return ids("id", id), err
				}
				return nil, err
			}
			want := instanceRunning
			if !start {
				want = instanceStopped
			}
			return ids("id", id), waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, want)
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Update: func(ctx context.Context, c *client.Client, p, s Values) error {
			id := s.Str("id")
			var paths []string
			body := map[string]any{"instanceId": id, "etag": s.Str("etag")}
			if p.Str("name") != s.Str("name") {
				paths = append(paths, "name")
				body["name"] = p.Str("name")
			}
			if p.Int("vcpus") != s.Int("vcpus") {
				paths = append(paths, "vcpus")
				body["vcpus"] = p.Int("vcpus")
			}
			if p.Int("memory_mib") != s.Int("memory_mib") {
				paths = append(paths, "memory_mib")
				body["memoryMib"] = p.Int("memory_mib")
			}
			if len(paths) > 0 {
				body["updateMask"] = fieldMask(paths...)
				op, err := c.Call(ctx, svcCompute, "UpdateInstance", body)
				if err != nil {
					return err
				}
				if _, err := c.WaitOperation(ctx, op); err != nil {
					return err
				}
			}
			want := !p.Has("start") || p.Bool("start")
			had := !s.Has("start") || s.Bool("start")
			if want != had {
				method, target := "StopInstance", instanceStopped
				if want {
					method, target = "StartInstance", instanceRunning
				}
				op, err := c.Call(ctx, svcCompute, method, map[string]any{"instanceId": id})
				if err != nil {
					return err
				}
				if _, err := c.WaitOperation(ctx, op); err != nil {
					return err
				}
				return waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, target)
			}
			return nil
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			op, err := c.Call(ctx, svcCompute, "DeleteInstance", map[string]any{"instanceId": s.Str("id"), "forceStop": true})
			if err != nil {
				return gone(err)
			}
			if _, err := c.WaitOperation(ctx, op); err != nil {
				return err
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

var templateReady = states{path: "status", ready: []string{"TEMPLATE_STATUS_READY"}, failed: []string{"TEMPLATE_STATUS_ERROR"}}

func capture(name, desc, service, kind, idField string) *Resource {
	get := func(ctx context.Context, c *client.Client, id string) (map[string]any, error) {
		obj, err := c.Call(ctx, service, "Get"+kind, map[string]any{"name": id})
		return obj, gone(err)
	}
	return &Resource{
		Name: name,
		Desc: desc,
		Attrs: []Attr{
			{Name: "id", Kind: String, Computed: true, Stable: true, Path: "name", Desc: "The full name."},
			{Name: idField, Kind: String, Required: true, ForceNew: true, Local: true, Desc: "The short name: lowercase letters, numbers and hyphens."},
			{Name: "instance_id", Kind: String, Required: true, ForceNew: true, Local: true, Desc: "The instance it is made from."},
			{Name: "display_name", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true},
			{Name: "description", Kind: String, Optional: true, Computed: true, Stable: true, ForceNew: true},
			{Name: "status", Kind: String, Computed: true, Enum: "TEMPLATE_STATUS_", Desc: "creating, ready, deleting or error."},
			{Name: "region_code", Kind: String, Computed: true, Stable: true},
			{Name: "zone_code", Kind: String, Computed: true, Stable: true},
			{Name: "series_code", Kind: String, Computed: true, Stable: true},
			{Name: "vcpus", Kind: Int, Computed: true, Stable: true},
			{Name: "memory_mib", Kind: Int, Computed: true, Stable: true},
			createdAttr("createTime"),
		},
		Create: func(ctx context.Context, c *client.Client, p Values) (Values, error) {
			body := map[string]any{camel(idField): p.Str(idField), "instanceId": p.Str("instance_id")}
			set(body, "displayName", p.Str("display_name"))
			set(body, "description", p.Str("description"))
			obj, err := c.Call(ctx, service, "Create"+kind, body)
			if err != nil {
				return nil, err
			}
			id := str(obj, "name")
			return ids("id", id), waitState(ctx, func() (map[string]any, error) { return get(ctx, c, id) }, templateReady)
		},
		Read: func(ctx context.Context, c *client.Client, s Values) (map[string]any, error) {
			return get(ctx, c, s.Str("id"))
		},
		Delete: func(ctx context.Context, c *client.Client, s Values) error {
			if _, err := c.Call(ctx, service, "Delete"+kind, map[string]any{"name": s.Str("id")}); err != nil {
				return gone(err)
			}
			return waitGone(ctx, func() (map[string]any, error) { return get(ctx, c, s.Str("id")) })
		},
	}
}

func templateResource() *Resource {
	return capture("template", "A template of an instance (its shape, disks and interfaces) to make instances from.", svcTemplate, "Template", "template_id")
}

func snapshotResource() *Resource {
	return capture("snapshot", "A snapshot of an instance.", svcSnapshot, "Snapshot", "snapshot_id")
}
