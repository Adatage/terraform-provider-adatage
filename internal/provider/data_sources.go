package provider

import (
	"context"
	"errors"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
)

func imageDataSource() *DataSource {
	return &DataSource{
		Name: "image",
		Desc: "Finds an image by name, or the newest public image of an operating system and version.",
		Attrs: []Attr{
			{Name: "id", Kind: String, Computed: true, Path: "name"},
			{Name: "name", Kind: String, Optional: true, Computed: true, Desc: "The name disks refer to."},
			{Name: "os", Kind: String, Optional: true, Computed: true},
			{Name: "os_version", Kind: String, Optional: true, Computed: true},
			{Name: "architecture", Kind: String, Optional: true, Computed: true},
			{Name: "display_name", Kind: String, Computed: true},
			{Name: "min_size_bytes", Kind: Int, Computed: true},
			{Name: "is_public", Kind: Bool, Computed: true},
		},
		Read: func(ctx context.Context, c *client.Client, cfg Values) (map[string]any, error) {
			list, err := c.Call(ctx, svcImage, "ListImages", map[string]any{})
			if err != nil {
				return nil, err
			}
			var found map[string]any
			for _, it := range asList(list["images"]) {
				m, ok := it.(map[string]any)
				if !ok || str(m, "status") != "IMAGE_STATUS_READY" {
					continue
				}
				if cfg.Str("name") != "" && str(m, "name") != cfg.Str("name") {
					continue
				}
				if cfg.Str("os") != "" && str(m, "os") != cfg.Str("os") {
					continue
				}
				if cfg.Str("os_version") != "" && str(m, "osVersion") != cfg.Str("os_version") {
					continue
				}
				if cfg.Str("architecture") != "" && str(m, "architecture") != cfg.Str("architecture") {
					continue
				}
				if found == nil || str(m, "createTime") > str(found, "createTime") {
					found = m
				}
			}
			if found == nil {
				return nil, errors.New("no image matches")
			}
			return found, nil
		},
	}
}

func networkDataSource() *DataSource {
	return &DataSource{
		Name: "network",
		Desc: "Finds a network by name or id.",
		Attrs: []Attr{
			{Name: "id", Kind: String, Optional: true, Computed: true, Path: "networkId"},
			{Name: "name", Kind: String, Optional: true, Computed: true, Path: "metadata.name"},
			{Name: "mtu", Kind: Int, Computed: true},
			{Name: "ipv6_enabled", Kind: Bool, Computed: true},
			{Name: "state", Kind: String, Computed: true, Enum: "PHASE_STATE_"},
		},
		Read: func(ctx context.Context, c *client.Client, cfg Values) (map[string]any, error) {
			if cfg.Str("id") == "" && cfg.Str("name") == "" {
				return nil, errors.New("set id or name")
			}
			list, err := c.Call(ctx, svcNetwork, "ListNetworks", map[string]any{})
			if err != nil {
				return nil, err
			}
			for _, it := range asList(list["networks"]) {
				m, ok := it.(map[string]any)
				if ok && (cfg.Str("id") == "" || str(m, "networkId") == cfg.Str("id")) && (cfg.Str("name") == "" || str(m, "metadata.name") == cfg.Str("name")) {
					return m, nil
				}
			}
			return nil, errors.New("no network matches")
		},
	}
}

func subnetDataSource() *DataSource {
	return &DataSource{
		Name: "subnet",
		Desc: "Finds a subnet by name or id.",
		Attrs: []Attr{
			{Name: "id", Kind: String, Optional: true, Computed: true, Path: "subnetId"},
			{Name: "name", Kind: String, Optional: true, Computed: true, Path: "metadata.name"},
			{Name: "network_id", Kind: String, Optional: true, Computed: true},
			{Name: "ipv4_cidr", Kind: String, Computed: true},
			{Name: "ipv6_cidr", Kind: String, Computed: true},
			{Name: "ip_type", Kind: String, Computed: true, Path: "type", Enum: "IP_TYPE_"},
		},
		Read: func(ctx context.Context, c *client.Client, cfg Values) (map[string]any, error) {
			if cfg.Str("id") == "" && cfg.Str("name") == "" {
				return nil, errors.New("set id or name")
			}
			list, err := c.Call(ctx, svcNetwork, "ListSubnets", map[string]any{})
			if err != nil {
				return nil, err
			}
			for _, it := range asList(list["subnets"]) {
				m, ok := it.(map[string]any)
				if ok && (cfg.Str("id") == "" || str(m, "subnetId") == cfg.Str("id")) && (cfg.Str("name") == "" || str(m, "metadata.name") == cfg.Str("name")) &&
					(cfg.Str("network_id") == "" || str(m, "networkId") == cfg.Str("network_id")) {
					return m, nil
				}
			}
			return nil, errors.New("no subnet matches")
		},
	}
}

func clusterCredentialsDataSource() *DataSource {
	return &DataSource{
		Name: "kubernetes_cluster_credentials",
		Desc: "The kubeconfig and the CA certificate of a cluster, for the kubernetes and helm providers. The kubeconfig signs in through adatagecli.",
		Attrs: []Attr{
			{Name: "id", Kind: String, Computed: true, Path: "clusterId"},
			{Name: "cluster_id", Kind: String, Required: true},
			{Name: "endpoint", Kind: String, Computed: true},
			{Name: "ca_certificate", Kind: String, Computed: true},
			{Name: "kubeconfig", Kind: String, Computed: true, Sensitive: true},
		},
		Read: func(ctx context.Context, c *client.Client, cfg Values) (map[string]any, error) {
			obj, err := c.Call(ctx, svcKubernetes, "GetClusterCredentials", map[string]any{"clusterId": cfg.Str("cluster_id")})
			if err != nil {
				return nil, err
			}
			obj["clusterId"] = cfg.Str("cluster_id")
			return obj, nil
		},
	}
}

func allDataSources() []*DataSource {
	return []*DataSource{imageDataSource(), networkDataSource(), subnetDataSource(), clusterCredentialsDataSource()}
}
