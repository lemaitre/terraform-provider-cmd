package cmd

import (
  "context"
  "fmt"
  "os/exec"

  "github.com/hashicorp/terraform-plugin-framework/diag"
  "github.com/hashicorp/terraform-plugin-framework/tfsdk"
  "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
  _ shell = shellLocal{}
)

var shellLocalFactory shellFactory = shellFactory{
  IsRemote: false,
  Name: "local",
  Schema: map[string]tfsdk.Attribute{
    "unused": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Unused",
      Optional: true,
    },
  },
  Create: func (_ context.Context, _ types.Object) (shell, diag.Diagnostics) {
    return shellLocal{}, nil
  },
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
