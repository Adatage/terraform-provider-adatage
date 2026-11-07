terraform {
  required_providers {
    adatage = {
      source = "adatage/adatage"
    }
  }
}

# Credentials: the JSON file of a service account (console > IAM > Service accounts), in ADATAGE_CREDENTIALS_FILE or here.
provider "adatage" {
  # credentials_file = "adatage-credentials.json"
  # project          = "149005782980031183"
}

resource "adatage_network" "main" {
  name = "main"
}

resource "adatage_subnet" "app" {
  network_id = adatage_network.main.id
  name       = "app"
  ip_type    = "ipv4_only"
  ipv4_cidr  = "10.20.0.0/24"
}

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
}

resource "adatage_ssh_key" "admin" {
  name       = "admin"
  public_key = file("~/.ssh/id_ed25519.pub")
}

data "adatage_image" "debian" {
  os         = "debian"
  os_version = "13"
}

resource "adatage_instance" "web" {
  name          = "web"
  region_code   = "eu-central-01"
  zone_code     = "a"
  series_code   = "general"
  vcpus         = 2
  memory_mib    = 4096
  ssh_key_names = [adatage_ssh_key.admin.name]

  disks = [{
    is_boot     = true
    auto_delete = true
    name        = "web-boot"
    size_gib    = 20
    image       = data.adatage_image.debian.name
  }]

  nics = [{ subnet_id = adatage_subnet.app.id }]
}

resource "adatage_public_ip" "web" {
  name        = "web"
  region_code = "eu-central-01"
  subnet_id    = adatage_subnet.app.id
  private_ipv4 = adatage_instance.web.primary_ipv4
}

output "web_public_address" {
  value = adatage_public_ip.web.address
}
