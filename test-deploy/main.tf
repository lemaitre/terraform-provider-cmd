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
    password = "dummy"
  }
  inputs = {
    dummy = "e"
  }

  create {
    cmd = "echo create; sleep 10"
  }
  update {
    cmd = "echo update; sleep 10"
  }
  destroy {
    cmd = "echo destroy; sleep 10"
  }
  reload {
    name = "a"
    cmd = "echo reload; sleep 10"
  }
}

#output "pouet" {
#  value = {
#    inputs = cmd_local.pouet.inputs
#    state = cmd_local.pouet.state
#  }
#}
