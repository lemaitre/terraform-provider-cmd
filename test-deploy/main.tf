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
    b = 3
    c = null_resource.dummy.id
  }
  create {
    cmd = "export"
  }
  destroy {
    cmd = "export"
  }

  update {
    triggers = ["a", "b"]
    cmd = "export"
  }
  update {
    triggers = ["b", "c"]
    cmd = "export"
  }
  
  reload {
    name = "a"
    cmd = "echo -n $INPUT_a"
  }
}

output "pouet" {
  value = {
    inputs = cmd_local.pouet.inputs
    state = cmd_local.pouet.state
  }
}
