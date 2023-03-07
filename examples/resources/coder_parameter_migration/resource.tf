provider "coder" {}

variable "old_account_name" {
  type    = string
  default = "fake-user" # for testing purposes, no need to set via env TF_...
}

data "coder_parameter" "account_name" {
  name                 = "Account Name"
  type                 = "string"
  legacy_variable_name = var.old_account_name
}