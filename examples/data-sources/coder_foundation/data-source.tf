terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

provider "coder" {}

# Coder resolves the current Foundation automatically in both provisioning scopes.
data "coder_foundation" "current" {}

locals {
  # A snapshot might not be available during template import.
  subnet_ids = try(data.coder_foundation.current.outputs.subnet_ids, toset([]))
}

output "foundation_id" {
  value = data.coder_foundation.current.id
}

output "snapshot_id" {
  value = data.coder_foundation.current.snapshot_id
}

# Public outputs stay usable in for_each even when the snapshot contains secrets.
resource "terraform_data" "subnet" {
  for_each = local.subnet_ids
  input    = each.value
}

output "subnet_ids" {
  value = local.subnet_ids
}

output "token" {
  value     = try(data.coder_foundation.current.sensitive_outputs.token, null)
  sensitive = true
}
