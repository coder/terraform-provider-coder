provider "coder" {}

# presets can be used to predefine common configurations for workspaces
# Parameters are referenced by their name. Each parameter must be defined in the preset.
# Values defined by the preset must pass validation for the parameter.
# See the coder_parameter data source's documentation for examples of how to define
# parameters like the ones used below.
data "coder_workspace_preset" "example" {
  name = "example"
  parameters = {
    (data.coder_parameter.example.name) = "us-central1-a"
    (data.coder_parameter.ami.name)     = "ami-xxxxxxxx"
  }
}

# Example of a default preset that will be pre-selected for users
data "coder_workspace_preset" "standard" {
  name    = "Standard"
  default = true
  parameters = {
    (data.coder_parameter.instance_type.name) = "t3.medium"
    (data.coder_parameter.region.name)        = "us-west-2"
  }
}
