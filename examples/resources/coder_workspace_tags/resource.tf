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

data "coder_workspace_tags" "custom_workspace_tags" {
  tag {
    name  = "cluster"
    value = "developers"
  }
  tag {
    name  = "os"
    value = data.coder_parameter.os_selector
  }
  tag {
    name  = "cache"
    value = data.coder_parameter.feature_cache_enabled == "true" ? "nix-with-cache" : "no-cache"
  }
}