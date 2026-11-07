package provider

var Examples = map[string]string{
	"resource.ssh_key": `
resource "adatage_ssh_key" "admin" {
  name       = "admin"
  public_key = file("~/.ssh/id_ed25519.pub")
}`,
	"resource.network": `
resource "adatage_network" "main" {
  name = "main"
}`,
	"resource.subnet": `
resource "adatage_subnet" "app" {
  network_id = adatage_network.main.id
  name       = "app"
  ip_type    = "ipv4_only"
  ipv4_cidr  = "10.20.0.0/24"
}`,
	"resource.firewall_rule": `
resource "adatage_firewall_rule" "ssh" {
  network_id  = adatage_network.main.id
  name        = "ssh"
  direction   = "ingress"
  action      = "allow"
  protocol    = "tcp"
  remote_cidr = "0.0.0.0/0"
  port_start  = 22
  port_end    = 22
  priority    = 100
}`,
	"resource.public_ip": `
resource "adatage_public_ip" "web" {
  name        = "web"
  region_code = "eu-central-01"

  # translated 1:1 to a private address of an instance (or of a load balancer) in the subnet
  subnet_id    = adatage_subnet.app.id
  private_ipv4 = adatage_instance.web.primary_ipv4
}`,
	"resource.disk": `
resource "adatage_disk" "data" {
  zone     = "eu-central-01a"
  disk_id   = "data"
  disk_type = "volumes-standard"
  size_gib  = 100
}`,
	"resource.instance": `
data "adatage_image" "debian" {
  name = "debian-13-genericcloud-amd64"
}

resource "adatage_instance" "web" {
  name        = "web"
  region_code = "eu-central-01"
  zone_code   = "a"
  series_code = "A"
  vcpus       = 2
  memory_mib  = 4096

  ssh_key_names = [adatage_ssh_key.admin.name]

  disks = [{
    is_boot     = true
    auto_delete = true
    name         = "web-boot"
    storage_pool = "volumes-standard"
    size_gib     = 20
    image        = data.adatage_image.debian.name
  }]

  nics = [{
    subnet_id = adatage_subnet.app.id
  }]
}`,
	"resource.template": `
resource "adatage_template" "web" {
  template_id  = "web"
  instance_id  = adatage_instance.web.id
  display_name = "Web server"
}`,
	"resource.snapshot": `
resource "adatage_snapshot" "before_upgrade" {
  snapshot_id = "before-upgrade"
  instance_id = adatage_instance.web.id
}`,
	"resource.kubernetes_cluster": `
resource "adatage_kubernetes_cluster" "main" {
  name        = "main"
  region_code = "eu-central-01"
  zone_code   = "a"
  subnet_id   = adatage_subnet.app.id

  node_pool = {
    name        = "default"
    series_code = "A"
    vcpus       = 2
    memory_mib  = 4096
    size        = 3
  }
}

data "adatage_kubernetes_cluster_credentials" "main" {
  cluster_id = adatage_kubernetes_cluster.main.id
}`,
	"resource.kubernetes_node_pool": `
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
}`,
	"resource.load_balancer_health_check": `
resource "adatage_load_balancer_health_check" "http" {
  name         = "http"
  protocol     = "http"
  port         = 80
  request_path = "/healthz"
}`,
	"resource.load_balancer_backend_service": `
resource "adatage_load_balancer_backend_service" "web" {
  name            = "web"
  health_check_id = adatage_load_balancer_health_check.http.id
  affinity        = "client_ip"

  backends = [{ instance_id = adatage_instance.web.id }]
}`,
	"resource.load_balancer_forwarding_rule": `
resource "adatage_load_balancer_forwarding_rule" "web" {
  name               = "web"
  region_code        = "eu-central-01"
  subnet_id          = adatage_subnet.app.id
  protocol           = "tcp"
  port_from          = 80
  port_to            = 443
  backend_service_id = adatage_load_balancer_backend_service.web.id
}

# a public address 1:1 translated to the private address of the load balancer
resource "adatage_public_ip" "web_lb" {
  name         = "web-lb"
  region_code  = "eu-central-01"
  subnet_id    = adatage_subnet.app.id
  private_ipv4 = adatage_load_balancer_forwarding_rule.web.address
}`,
	"resource.database_instance": `
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
}`,
	"resource.database_backup": `
resource "adatage_database_backup" "before_upgrade" {
  instance_id = adatage_database_instance.main.id
  description = "before the upgrade"
}`,
	"data.database_engine_versions": `
data "adatage_database_engine_versions" "valkey" {
  engine = "valkey"
}`,
	"resource.iam_role": `
resource "adatage_iam_role" "viewer" {
  name        = "Viewer"
  permissions = ["compute.instances.list", "compute.instances.get", "network.networks.list"]
}`,
	"resource.service_account": `
resource "adatage_service_account" "ci" {
  name = "ci"
  role = "admin"
}`,
	"resource.service_account_key": `
resource "adatage_service_account_key" "ci" {
  service_account_id = adatage_service_account.ci.id
}

output "ci_credentials" {
  value     = adatage_service_account_key.ci.credentials
  sensitive = true
}`,
	"data.image": `
data "adatage_image" "debian" {
  os         = "debian"
  os_version = "13"
}`,
	"data.network": `
data "adatage_network" "main" {
  name = "main"
}`,
	"data.subnet": `
data "adatage_subnet" "app" {
  name = "app"
}`,
	"data.kubernetes_cluster_credentials": `
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
}`,
}
