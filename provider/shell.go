package provider

import (
  "bytes"
  "fmt"
  "os/exec"
)

type shell interface {
  Execute(string, map[string]string) (string, string, error)
  //Send(string, []byte) error
  //Receive(string) ([]byte, error)
  Destroy()
}

type shell_local struct {
  args []string
}

func (sh shell_local) Execute(command string, env map[string]string) (stdout string, stderr string, err error) {
  if len(sh.args) == 0 {
    sh.args = []string{"sh", "-c", command}
  } else {
    sh.args = append(sh.args, command)
  }

  cmd := exec.Command(sh.args[0], sh.args[1:]...)

  for k, v := range env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
  }

  var outb, errb bytes.Buffer
  cmd.Stdout = &outb
  cmd.Stderr = &errb

  err = cmd.Run()

  stdout = outb.String()
  stderr = errb.String()

  return
}
func (_ shell_local) Destroy() {}
