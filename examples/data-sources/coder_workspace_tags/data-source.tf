provider "coder" {}

data "coder_parameter" "os_selector" {
  name         = "os_selector"
  display_name = "Operating System"
  mutable      = false

  default = "osx"

  option {
    icon  = "/icons/linux.png"
    name  = "Linux"
    value = "linux"
  }
  option {
    icon  = "/icons/osx.png"
    name  = "OSX"
    value = "osx"
  }
  option {
    icon  = "/icons/windows.png"
    name  = "Windows"
    value = "windows"
  }
}

data "coder_parameter" "feature_cache_enabled" {
  name         = "feature_cache_enabled"
  display_name = "Enable cache?"
  type         = "bool"

  default = false
}

data "coder_parameter" "feature_debug_enabled" {
  name         = "feature_debug_enabled"
  display_name = "Enable debug?"
  type         = "bool"

  default = true
}

data "coder_workspace_tags" "custom_workspace_tags" {
  tags = {
    "cluster" = "developers"
    "os"      = data.coder_parameter.os_selector.value
    "debug"   = "${data.coder_parameter.feature_debug_enabled.value}+12345"
    "cache"   = data.coder_parameter.feature_cache_enabled.value == "true" ? "nix-with-cache" : "no-cache"
  }
}