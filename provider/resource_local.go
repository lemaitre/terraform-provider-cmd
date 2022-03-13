package provider

import (
  "context"
  "strings"
  "sort"
  "fmt"

  "github.com/hashicorp/terraform-plugin-framework/diag"
  "github.com/hashicorp/terraform-plugin-framework/tfsdk"
  "github.com/hashicorp/terraform-plugin-framework/types"
  "github.com/hashicorp/terraform-plugin-go/tftypes"
  "github.com/hashicorp/terraform-plugin-log/tflog"
)

type cmdLocalType struct{}

// GetSchema returns the Terraform Schema of the cmd_local resource.
func (t cmdLocalType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
  return tfsdk.Schema{
    Description: "Custom resource managed by local shell scripts",
    MarkdownDescription: "Custom resource managed by local shell scripts",

    Attributes: map[string]tfsdk.Attribute{
      "inputs": {
        Required:            true,
        MarkdownDescription: "Inputs",
        PlanModifiers: tfsdk.AttributePlanModifiers{
          planModifier{},
        },
        Type: types.MapType{types.StringType},
      },
      "state": {
        Computed:            true,
        MarkdownDescription: "State",
        PlanModifiers: tfsdk.AttributePlanModifiers{
          tfsdk.UseStateForUnknown(),
        },
        Type: types.MapType{types.StringType},
      },
      "id": {
        Computed:            true,
        MarkdownDescription: "Example identifier",
        PlanModifiers: tfsdk.AttributePlanModifiers{
          tfsdk.UseStateForUnknown(),
        },
        Type: types.StringType,
      },
    },

    Blocks: map[string]tfsdk.Block{
      "update": {
        NestingMode: tfsdk.BlockNestingModeList,
        Attributes: map[string]tfsdk.Attribute{
          "triggers": {
            MarkdownDescription: "What variable changes trigger the update",
            Optional:            true,
            Type:                types.SetType{types.StringType},
          },
          "cmd": {
            MarkdownDescription: "Command to execute",
            Required:            true,
            Type:                types.StringType,
          },
        },
      },
      "reload": {
        NestingMode: tfsdk.BlockNestingModeList,
        Attributes: map[string]tfsdk.Attribute{
          "name": {
            MarkdownDescription: "Variable name to reload",
            Required:            true,
            Type:                types.StringType,
          },
          "cmd": {
            MarkdownDescription: "Command to execute",
            Required:            true,
            Type:                types.StringType,
          },
        },
      },
      "create": {
        NestingMode: tfsdk.BlockNestingModeList,
        MinItems: 0,
        MaxItems: 1,
        Attributes: map[string]tfsdk.Attribute{
          "cmd": {
            MarkdownDescription: "Command to execute",
            Required:            true,
            Type:                types.StringType,
          },
        },
      },
      "destroy": {
        NestingMode: tfsdk.BlockNestingModeList,
        MinItems: 0,
        MaxItems: 1,
        Attributes: map[string]tfsdk.Attribute{
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

// NewResource creates a resource handle to a cmd_local resource.
func (t cmdLocalType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
  provider, diags := convertProviderType(in)

  return cmdLocal{
    provider: provider,
  }, diags
}

// cmdLocalData encodes the data of a cmd_local resource.
type cmdLocalData struct {
  Id   types.String `tfsdk:"id"`
  Input map[string]types.String `tfsdk:"inputs"`
  State map[string]string `tfsdk:"state"`
  Reload []struct {
    Name string `tfsdk:"name"`
    Cmd string `tfsdk:"cmd"`
  } `tfsdk:"reload"`
  Update []struct {
    Triggers []string `tfsdk:"triggers"`
    Cmd string `tfsdk:"cmd"`
  } `tfsdk:"update"`
  Create []struct {
    Cmd string `tfsdk:"cmd"`
  } `tfsdk:"create"`
  Destroy []struct {
    Cmd string `tfsdk:"cmd"`
  } `tfsdk:"destroy"`
}

// cmdLocal is a resource handle to a cmd_local resource.
type cmdLocal struct {
  provider provider
}

// Create is in charge to crete a cmd_local resource.
func (r cmdLocal) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
  var data cmdLocalData

  diags := req.Config.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  type void struct {}
  seen := make(map[string]void)
  conflict := make(map[string]void)

  for _, update0 := range data.Update {
    triggers0 := strings.Join(update0.Triggers, ",")
    if _, found := seen[triggers0]; found {
      resp.Diagnostics.AddError("Update ambiguity", fmt.Sprintf("Update rule for %s is duplicated", triggers0))
    }
    seen[triggers0] = void{}

    for _, update1 := range data.Update {
      left, inner, right := sorted_list_3way(update0.Triggers, update1.Triggers)
      if len(left) > 0 && len(inner) > 0 && len(right) > 0 {
        conflict[strings.Join(inner, ",")] = void{}
      }
    }
  }

  for k := range conflict {
    if _, found := seen[k]; !found {
      resp.Diagnostics.AddError("Update Ambiguity", fmt.Sprintf("Update of %s would lead to ambiguous update rule", k))
    }
  }

  if resp.Diagnostics.HasError() {
    return
  }

  for _, create := range data.Create {
    cmd := create.Cmd
    stdout, stderr, err := execute("sh", "-c", cmd)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, "cmd", cmd)
    }
    if len(stdout) > 0 {
      tflog.Info(ctx, stdout, "cmd", cmd)
    }

    if err != nil {
      resp.Diagnostics.AddError("Command error", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, stderr))
      return
    }
  }

  data.State = map[string]string{}

  // For the purposes of this example code, hardcoding a response value to
  // save into the Terraform state.
  data.Id = types.String{Value: "example-id"}

  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to read the state of a cmd_local resource during a refresh.
func (r cmdLocal) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
  var data cmdLocalData
  tflog.Warn(ctx, "Read: start")

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    tflog.Warn(ctx, "Read: error")
    return
  }

  // If applicable, this is a great opportunity to initialize any necessary
  // provider client data and make a call using it.
  // example, err := d.provider.client.ReadExample(...)
  // if err != nil {
  //     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
  //     return
  // }

  tflog.Warn(ctx, "Read: set")
  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
  tflog.Warn(ctx, "Read: finished")
}

// Read is in charge to update a cmd_local resource.
func (r cmdLocal) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
  var plan, state cmdLocalData

  diags := req.Plan.Get(ctx, &plan)
  resp.Diagnostics.Append(diags...)
  diags = req.State.Get(ctx, &state)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  cmd := get_update_cmd(state.Input, plan.Input, state)

  if len(cmd) > 0 {
    stdout, stderr, err := execute("sh", "-c", cmd)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, "cmd", cmd)
    }
    if len(stdout) > 0 {
      tflog.Info(ctx, stdout, "cmd", cmd)
    }

    if err != nil {
      resp.Diagnostics.AddError("Command error", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, stderr))
      return
    }
  }

  diags = resp.State.Set(ctx, &plan)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to delete a cmd_local resource.
func (r cmdLocal) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
  var data cmdLocalData

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  for _, create := range data.Destroy {
    cmd := create.Cmd
    stdout, stderr, err := execute("sh", "-c", cmd)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, "cmd", cmd)
    }
    if len(stdout) > 0 {
      tflog.Info(ctx, stdout, "cmd", cmd)
    }

    if err != nil {
      resp.Diagnostics.AddError("Command error", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, stderr))
      return
    }
  }

  resp.State.RemoveResource(ctx)
}

// ImportState is in charge to import a cmd_local resource into terraform.
func (r cmdLocal) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
  tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

// get_update_cmd search for the right command to execute satisfying the update policies of the resource.
func get_update_cmd(state map[string]types.String, plan map[string]types.String, rules cmdLocalData) string {
  var modified []string
  for k, x := range state {
    if y, found := plan[k]; !found || x != y {
      modified = append(modified, k)
    }
  }
  for k, _ := range plan {
    if _, found := state[k]; !found {
      modified = append(modified, k)
    }
  }
  sort.Strings(modified)

  var trig []string
  cmd := ""

  for _, update := range rules.Update {
    triggers := update.Triggers
    if len(triggers) == 0 {
      if len(cmd) == 0 {
        cmd = update.Cmd
      }
    } else if len(triggers) >= len(modified) && (len(trig) == 0 || len(trig) > len(triggers)) {
      _, _, right := sorted_list_3way(triggers, modified)
      if len(right) == 0 {
        trig = triggers
        cmd = update.Cmd
      }
    }
  }

  return cmd
}

type planModifier struct {}

func (_ planModifier) Description(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ planModifier) MarkdownDescription(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ planModifier) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
  var plan, state cmdLocalData

  if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
    return
  }
  if req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() {
    return
  }

  diags := req.Config.Get(ctx, &plan)
  resp.Diagnostics.Append(diags...)

  diags = req.State.Get(ctx, &state)
  resp.Diagnostics.Append(diags...)

  cmd := get_update_cmd(state.Input, plan.Input, state)
  if len(cmd) == 0 {
    resp.RequiresReplace = true
  }
}


