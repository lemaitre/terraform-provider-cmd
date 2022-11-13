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
    dummy = md5("b")
  }

  create {
    cmd = "echo create"
  }
  update {
    cmd = "echo update"
    invalidates = ["a"]
  }
  destroy {
    cmd = "echo destroy"
  }
  reload {
    name = "a"
    cmd = <<-EOT
    echo -n "$INPUT_dummy"
    EOT
  }
  reload {
    name = "b"
    cmd = "echo plop"
  }
}

output "plop" {
  value = cmd_ssh.plop
}
