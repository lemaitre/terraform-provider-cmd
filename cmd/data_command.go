package cmd

import (
  "context"
  "fmt"
  "regexp"

  "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
  //"github.com/hashicorp/terraform-plugin-framework/attr"
  "github.com/hashicorp/terraform-plugin-framework/datasource"
  "github.com/hashicorp/terraform-plugin-framework/diag"
  //"github.com/hashicorp/terraform-plugin-framework/path"
  //"github.com/hashicorp/terraform-plugin-framework/provider"
  //"github.com/hashicorp/terraform-plugin-framework/resource"
  "github.com/hashicorp/terraform-plugin-framework/tfsdk"
  "github.com/hashicorp/terraform-plugin-framework/types"
  "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the desired interfaces.
var _ datasource.DataSource = &dataSourceCommand{}

type dataSourceCommand struct {
  shell shell
  shellFactory shellFactory
}

type dataSourceCommandModel struct {
  Input map[string]types.String `tfsdk:"inputs"`
  State map[string]types.String `tfsdk:"state"`
  ConnectionOptions types.Object `tfsdk:"connection"`
  Read []dataSourceCommandReadModel `tfsdk:"read"`
}
type dataSourceCommandReadModel struct {
  Name string `tfsdk:"name"`
  Cmd string `tfsdk:"cmd"`
}

func (d *dataSourceCommand) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
  resp.TypeName = req.ProviderTypeName + "_" + d.shellFactory.Name
}

func (d *dataSourceCommand) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
  return tfsdk.Schema{
    Description: "Custom resource managed by local shell scripts",
    MarkdownDescription: "Custom resource managed by local shell scripts",

    Attributes: map[string]tfsdk.Attribute{
      "connection": {
        Optional:            true,
        MarkdownDescription: "Connection Options",
        Attributes: tfsdk.SingleNestedAttributes(d.shellFactory.Schema),
      },
      "inputs": {
        Required:            true,
        MarkdownDescription: "Inputs",
        Type: types.MapType{types.StringType},
      },
      "state": {
        Computed:            true,
        MarkdownDescription: "State",
        Type: types.MapType{types.StringType},
      },
    },

    Blocks: map[string]tfsdk.Block{
      "read": {
        NestingMode: tfsdk.BlockNestingModeSet,
        Attributes: map[string]tfsdk.Attribute{
          "name": {
            MarkdownDescription: "Variable name to reload",
            Required:            true,
            Type:                types.StringType,
            Validators: []tfsdk.AttributeValidator{
              stringvalidator.LengthAtLeast(1),
              stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z]\w*$`), "must start with a letter and contain only letters, digits and underscore"),
            },
          },
          "cmd": {
            MarkdownDescription: "Command to execute",
            Required:            true,
            Type:                types.StringType,
          },
        },
      },
    },
  }, nil
}

func (d *dataSourceCommand) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
  var data dataSourceCommandModel

  // Read Terraform configuration data into the model
  resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

  if resp.Diagnostics.HasError() {
    return
  }

  if d.shell == nil {
    sh, diags := d.shellFactory.Create(ctx, data.ConnectionOptions)
    if len(diags) > 0 {
      resp.Diagnostics.Append(diags...)
      return
    }
    d.shell = sh
  }

  data.State = make(map[string]types.String)

  env := make(map[string]string)
  for k, v := range data.Input {
    env[fmt.Sprintf("INPUT_%s", k)] = v.ValueString()
  }

  for _, read := range data.Read {
    name := read.Name
    cmd := read.Cmd

    stdout, stderr, _, err := d.shell.Execute(cmd, env)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, map[string]any{"cmd": cmd})
    }
    if err == nil {
      data.State[name] = types.StringValue(stdout)
    } else {
      resp.Diagnostics.AddError("Command error during reading", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, stderr))
    }
  }

  // Save data into Terraform state
  resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
