terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

provider "coder" {}

# Proposed POC data source, requiring the companion development provider.
# Coder supplies an immutable build snapshot with sensitive outputs excluded.
data "coder_template_outputs" "shared" {}

locals {
  template_outputs = data.coder_template_outputs.shared.outputs

  # Template import can run before shared outputs exist. Pick a fallback
  # appropriate for the consuming resource, rather than indexing directly.
  subnet_ids = try(local.template_outputs.network.subnet_ids, [])
}

output "subnet_ids" {
  value = local.subnet_ids
}

output "template_resource_revision_id" {
  value = data.coder_template_outputs.shared.revision_id
}
