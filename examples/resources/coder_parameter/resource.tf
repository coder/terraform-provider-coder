data "coder_parameter" "example" {
  name        = "Region"
  description = "Specify a region to place your workspace."
  mutable     = false
  type        = "string"
  option {
    value = "us-central1-a"
    name  = "US Central"
    icon  = "/icon/usa.svg"
  }
  option {
    value = "asia-central1-a"
    name  = "Asia"
    icon  = "/icon/asia.svg"
  }
}

data "coder_parameter" "ami" {
  name = "Machine Image"
  option {
    value = "ami-xxxxxxxx"
    name  = "Ubuntu"
    icon  = "/icon/ubuntu.svg"
  }
}

data "coder_parameter" "image" {
  name = "Docker Image"
  icon = "/icon/docker.svg"
  type = "bool"
}

data "coder_parameter" "cores" {
  name = "CPU Cores"
  icon = "/icon/"
}

data "coder_parameter" "disk_size" {
  name = "Disk Size"
  type = "number"
  validation {
    # This can apply to number and string types.
    min = 0
    max = 10
  }
}
