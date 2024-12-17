terraform {
  required_providers {
    dns = {
      source = "hashicorp/dns"
      version = "3.4.2"
    }
  }
}

provider "dns" {
}