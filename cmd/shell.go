package cmd

import (
  "fmt"
  "os/exec"
)

type shell interface {
  Execute(string, map[string]string) (string, string, string, error)
  //Send(string, []byte) error
  //Receive(string) ([]byte, error)
  Close()
}

type shellLocal struct {
  args []string
}

func (sh shellLocal) Execute(command string, env map[string]string) (string, string, string, error) {
  if len(sh.args) == 0 {
    sh.args = []string{"sh", "-c", command}
  } else {
    sh.args = append(sh.args, command)
  }

  cmd := exec.Command(sh.args[0], sh.args[1:]...)

  for k, v := range env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
  }

  out := NewCommandOutput()

  cmd.Stdout = out.StdoutWriter
  cmd.Stderr = out.StderrWriter

  err := cmd.Run()

  return out.Stdout.String(), out.Stderr.String(), out.Combined.String(), err
}
func (_ shellLocal) Close() {}

func shellLocalFactory(map[string]string) (shell, error) {
  return shellLocal{}, nil
}
