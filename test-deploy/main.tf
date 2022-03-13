terraform {
  required_providers {
    cmd = {
      source  = "lemaitre.re/lemaitre/cmd"
      version = "~> 0.1"
    }
  }
}


provider "cmd" {
}

resource "null_resource" "dummy" {
}

resource "cmd_local" "pouet" {
  inputs = {
    a = 2
    b = 1
    c = null_resource.dummy.id
  }
  create {
    cmd = "echo Created"
  }
  destroy {
    cmd = "echo Destroyed >&2"
  }

  update {
    triggers = ["a"]
    cmd = "env"
  }
  update {
    triggers = ["a", "b"]
    cmd = "env"
  }
}

output "pouet" {
  value = cmd_local.pouet
}
