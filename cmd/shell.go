package cmd

import (
  "context"

  "github.com/hashicorp/terraform-plugin-framework/diag"
  "github.com/hashicorp/terraform-plugin-framework/tfsdk"
  "github.com/hashicorp/terraform-plugin-framework/types"
)

type shell interface {
  Execute(string, map[string]string) (string, string, string, error)
  //Send(string, []byte) error
  //Receive(string) ([]byte, error)
  Close()
}

type shellFactory struct {
  IsRemote bool
  Name string
  Schema map[string]tfsdk.Attribute
  Create func(context.Context, types.Object) (shell, diag.Diagnostics)
}
