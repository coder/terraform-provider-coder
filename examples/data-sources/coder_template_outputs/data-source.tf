terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

provider "coder" {}

# Deprecated compatibility data source. Prefer coder_foundation for new templates.
# Coder supplies an immutable snapshot with Terraform types and sensitivity.
data "coder_template_outputs" "shared" {}

locals {
  template_outputs = data.coder_template_outputs.shared.outputs

  # Template import can run before shared resource outputs exist. Pick a
  # fallback appropriate for the consuming resource, rather than indexing directly.
  subnet_ids = try(local.template_outputs.network.subnet_ids, [])
}

output "subnet_ids" {
  value     = local.subnet_ids
  sensitive = true
}

output "template_resource_revision_id" {
  value = data.coder_template_outputs.shared.revision_id
}
