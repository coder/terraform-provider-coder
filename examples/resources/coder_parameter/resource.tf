data "coder_parameter" "example" {
  display_name = "Region"
  description  = "Specify a region to place your workspace."
  immutable    = true
  type         = "string"
  option {
    value = "us-central1-a"
    label = "US Central"
    icon  = "/icon/usa.svg"
  }
  option {
    value = "asia-central1-a"
    label = "Asia"
    icon  = "/icon/asia.svg"
  }
}

data "coder_parameter" "ami" {
  display_name = "Machine Image"
  option {
    value = "ami-xxxxxxxx"
    label = "Ubuntu"
    icon  = "/icon/ubuntu.svg"
  }
}

data "coder_parameter" "image" {
  display_name = "Docker Image"
  icon         = "/icon/docker.svg"
  type         = "bool"
}

data "coder_parameter" "cores" {
  display_name = "CPU Cores"
  icon         = "/icon/"
}

data "coder_parameter" "disk_size" {
  display_name = "Disk Size"
  type         = "number"
  validation {
    # This can apply to number and string types.
    min = 0
    max = 10
  }
}
