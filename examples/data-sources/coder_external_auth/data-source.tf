provider "coder" {}


data "coder_external_auth" "github" {
  id = "github"
}

data "coder_external_auth" "azure-identity" {
  id       = "azure-identiy"
  optional = true
}
