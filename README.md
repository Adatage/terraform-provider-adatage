# terraform-provider-adatage

Terraform and OpenTofu provider for Adatage Cloud: instances, disks, networks, subnets, firewall rules, public IPs, load balancers, Kubernetes clusters and node pools, roles and service accounts. Documentation for users is in `docs/` (generated, see below); how it works in the platform repository is in `docs/cloud-platform/terraform-provider.md`.

```hcl
terraform {
  required_providers {
    adatage = { source = "adatage/adatage" }
  }
}

provider "adatage" {
  credentials_file = "adatage-credentials.json" # a service account key, made in the console
}
```

See `examples/complete/main.tf` for a network, a subnet, a firewall rule, an instance and a public IP.

## Development

The module is standalone (it has its own `go.mod` and is **not** in `go.work`), because it is published on its own to a public repository: it must not import the platform's private modules. It talks to the API with plain Connect JSON (`internal/client`).

```bash
pnpm nx build terraform-provider-adatage   # go vet + go build
pnpm nx test terraform-provider-adatage    # unit tests against a fake API
pnpm nx run terraform-provider-adatage:docs # regenerate docs/ from the resource definitions
```

Resources are data: `internal/provider/resources_*.go` list the attributes (`Attr`) of each resource and four functions that call the API (`Create`, `Read`, `Update`, `Delete`); `resource.go` turns them into the Terraform schema and the plan/apply cycle, `attr.go` converts between Terraform values and the API's JSON. To add a resource, write its `Resource`, add it to `allResources()` and an example to `examples.go`, then regenerate the docs.

To try a build with Terraform or OpenTofu, point `dev_overrides` at `bin/`:

```hcl
provider_installation {
  dev_overrides { "adatage/adatage" = "/path/to/apps/terraform-provider-adatage/bin" }
  direct {}
}
```
(the binary must be named `terraform-provider-adatage`; `pnpm nx build` does that).

## Releasing

See `RELEASING.md`.
