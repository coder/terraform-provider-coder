provider "coder" {}

data "coder_parameter" "example" {
  name        = "Region"
  description = "Specify a region to place your workspace."
  mutable     = false
  type        = "string"
  default     = "us-central1-a"
  option {
    value = "us-central1-a"
    name  = "US Central"
    icon  = "/icons/1f1fa-1f1f8.png"
  }
  option {
    value = "asia-southeast1-a"
    name  = "Singapore"
    icon  = "/icons/1f1f8-1f1ec.png"
  }
}

data "coder_parameter" "ami" {
  name        = "Machine Image"
  description = <<-EOT
    # Provide the machine image
    See the [registry](https://container.registry.blah/namespace) for options.
    EOT
  option {
    value = "ami-xxxxxxxx"
    name  = "Ubuntu"
    icon  = "/icon/ubuntu.svg"
  }
}

data "coder_parameter" "is_public_instance" {
  name    = "Is public instance?"
  type    = "bool"
  icon    = "/icon/docker.svg"
  default = false
}

data "coder_parameter" "cores" {
  name    = "CPU Cores"
  type    = "number"
  icon    = "/icon/cpu.svg"
  default = 3
  order   = 10
}

data "coder_parameter" "disk_size" {
  name    = "Disk Size"
  type    = "number"
  default = "5"
  order   = 8
  validation {
    # This can apply to number.
    min       = 0
    max       = 10
    monotonic = "increasing"
  }
}

data "coder_parameter" "cat_lives" {
  name    = "Cat Lives"
  type    = "number"
  default = "9"
  validation {
    # This can apply to number.
    min       = 0
    max       = 10
    monotonic = "decreasing"
  }
}

data "coder_parameter" "fairy_tale" {
  name      = "Fairy Tale"
  type      = "string"
  mutable   = true
  default   = "Hansel and Gretel"
  ephemeral = true
}

data "coder_parameter" "users" {
  name         = "system_users"
  display_name = "System users"
  type         = "list(string)"
  default      = jsonencode(["root", "user1", "user2"])
}

data "coder_parameter" "home_volume_size" {
  name        = "Home Volume Size"
  description = <<-EOF
  How large should your home volume be?
  EOF
  type        = "number"
  default     = 30
  mutable     = true
  order       = 3

  option {
    name  = "30GB"
    value = 30
  }

  option {
    name  = "60GB"
    value = 60
  }

  option {
    name  = "100GB"
    value = 100
  }

  validation {
    monotonic = "increasing"
  }
}

data "coder_parameter" "tools" {
  name        = "Tools"
  description = "What tools do you want to install?"
  type        = "list(string)"
  form_type   = "multi-select"
  styling = jsonencode({
    # Arbitrary JSON object to be passed to the frontend
  })
  default = jsonencode(["git", "docker"])

  option {
    value = "Docker"
    name  = "docker"
    icon  = "/icon/docker.svg"
  }
  option {
    value = "Git"
    name  = "git"
    icon  = "/icon/git.svg"
  }
  option {
    value = "Golang"
    name  = "go"
    icon  = "/icon/go.svg"
  }
  option {
    value = "Typescript"
    name  = "ts"
    icon  = "/icon/typescript.svg"
  }
}