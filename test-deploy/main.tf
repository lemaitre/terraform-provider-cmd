terraform {
  required_providers {
    cmd = {
      source  = "lemaitre.re/lemaitre/cmd"
      version = "~> 0.1.0"
    }
  }
}


resource "null_resource" "dummy" {
}

data "cmd_ssh" "pouet" {
  connection = {
    hostname = "bender.csdt.fr"
    username = "dummy-user"
    keyfile = "dummy.rsa"
    pouet = null_resource.dummy.id
  }

  inputs = {
  }

  read {
    name = "a"
    cmd = <<-EOT
    echo -n "Pouet"
    EOT
  }

}

#resource "cmd_local" "pouet" {
#  inputs = {
#    a = 3
#    b = 4
#    c = null_resource.dummy.id
#  }
#  create {
#    cmd = "export"
#  }
#  destroy {
#    cmd = "export"
#  }
#
#  update {
#    triggers = ["b"]
#    cmd = "export"
#  }
#  update {
#    triggers = ["a", "b"]
#    cmd = "export"
#  }
#  update {
#    triggers = ["b", "c"]
#    cmd = "export"
#  }
#  
#  reload {
#    name = "a"
#    cmd = "echo -n $INPUT_a"
#  }
#}

resource "cmd_ssh" "plop" {
  connection = {
    hostname = "bender.csdt.fr"
    username = "dummy-user"
    keyfile = "dummy.rsa"
    pouet = null_resource.dummy.id
  }
  inputs = {
    dummy = md5("f")
  }

  create {
    cmd = "echo create"
  }
  update {
    cmd = "echo update"
    reloads = ["a", "b"]
  }
  destroy {
    cmd = "echo destroy"
  }
  read {
    name = "a"
    cmd = <<-EOT
    echo -n "$INPUT_dummy"
    EOT
  }
  read {
    name = "b"
    cmd = "echo pouet"
  }
}

output "pouet" {
  value = {
    inputs = data.cmd_ssh.pouet.inputs,
    state  = data.cmd_ssh.pouet.state,
  }
}
output "plop" {
  value = {
    inputs = cmd_ssh.plop.inputs,
    state  = cmd_ssh.plop.state,
  }
}
