package provider

import (
  "bytes"
  "os/exec"
)

func capture_all(cmd *exec.Cmd) (stdout string, stderr string, err error) {
  var outb, errb bytes.Buffer

  cmd.Stdout = &outb
  cmd.Stderr = &errb

  err = cmd.Run()

  stdout = outb.String()
  stderr = errb.String()

  return
}

func execute(bin string, args ...string) (string, string, error) {
  return capture_all(exec.Command(bin, args...))
}
