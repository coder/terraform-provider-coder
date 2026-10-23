terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

provider "coder" {}

# Proposed POC data source, requiring the companion development provider.
# Coder supplies template identity for template-scoped runs and workspace builds.
data "coder_template" "current" {}

output "template_id" {
  value = data.coder_template.current.id
}

output "template_name" {
  value = data.coder_template.current.name
}
