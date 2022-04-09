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

type resourceCmdType struct{
  shellFactory func(map[string]string) shell
}

// GetSchema returns the Terraform Schema of the cmd_local resource.
func (t resourceCmdType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
  return tfsdk.Schema{
    Description: "Custom resource managed by local shell scripts",
    MarkdownDescription: "Custom resource managed by local shell scripts",

    Attributes: map[string]tfsdk.Attribute{
      "connection": {
        Optional:            true,
        MarkdownDescription: "Connection Options",
        Type: types.MapType{types.StringType},
      },
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
        //PlanModifiers: tfsdk.AttributePlanModifiers{
        //  tfsdk.UseStateForUnknown(),
        //},
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
        NestingMode: tfsdk.BlockNestingModeSet,
        Attributes: map[string]tfsdk.Attribute{
          "triggers": {
            MarkdownDescription: "What variable changes trigger the update",
            Optional:            true,
            Type:                types.ListType{types.StringType},
          },
          "cmd": {
            MarkdownDescription: "Command to execute",
            Required:            true,
            Type:                types.StringType,
          },
        },
        Validators: []tfsdk.AttributeValidator{
          updateValidator{},
        },
      },
      "reload": {
        NestingMode: tfsdk.BlockNestingModeSet,
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
        NestingMode: tfsdk.BlockNestingModeSet,
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
        NestingMode: tfsdk.BlockNestingModeSet,
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
func (t resourceCmdType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
  provider, diags := convertProviderType(in)

  return &resourceCmd{
    provider: provider,
    shell: nil,
    shellFactory: t.shellFactory,
  }, diags
}

// resourceCmd is a resource handle to a cmd_local resource.
type resourceCmd struct {
  provider provider
  shell shell
  shellFactory func(map[string]string) shell
}

// resourceCmdData encodes the data of a cmd_local resource.
type resourceCmdData struct {
  Id   types.String `tfsdk:"id"`
  Input map[string]types.String `tfsdk:"inputs"`
  State map[string]string `tfsdk:"state"`
  ConnectionOptions map[string]string `tfsdk:"connection"`
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

func (r *resourceCmd) init(ctx context.Context, data resourceCmdData) {
  if r.shellFactory != nil {
    r.shell = r.shellFactory(data.ConnectionOptions)
    r.shellFactory = nil
  }
}

// Create is in charge to crete a cmd_local resource.
func (r *resourceCmd) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
  var data resourceCmdData

  diags := req.Config.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  if resp.Diagnostics.HasError() {
    return
  }

  r.init(ctx, data)

  for _, create := range data.Create {
    cmd := create.Cmd
    env := make(map[string]string)
    for k, v := range data.Input {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("INPUT_%s", k)] = s
    }
    stdout, stderr, err := r.shell.Execute(cmd, env)

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

  data.State = make(map[string]string)
  data.read_state(ctx, r.shell, true)

  // For the purposes of this example code, hardcoding a response value to
  // save into the Terraform state.
  data.Id = types.String{Value: generate_id()}

  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to read the state of a cmd_local resource during a refresh.
func (r *resourceCmd) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
  var data resourceCmdData

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  r.init(ctx, data)

  data.read_state(ctx, r.shell, true)

  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to update a cmd_local resource.
func (r *resourceCmd) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
  var plan, state resourceCmdData

  diags := req.State.Get(ctx, &plan)
  diags = req.Config.Get(ctx, &plan)
  //resp.Diagnostics.Append(diags...)
  diags = req.State.Get(ctx, &state)
  //resp.Diagnostics.Append(diags...)

  plan.Id = state.Id

  if plan.State == nil {
    plan.State = make(map[string]string)
  }

  if resp.Diagnostics.HasError() {
    return
  }

  r.init(ctx, plan)

  cmd := get_update_cmd(state.Input, plan.Input, state)

  if len(cmd) > 0 {
    env := make(map[string]string)

    for k, v := range plan.Input {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("INPUT_%s", k)] = s
    }
    for k, v := range state.Input {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("PREVIOUS_%s", k)] = s
    }
    stdout, stderr, err := r.shell.Execute(cmd, env)

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

  plan.read_state(ctx, r.shell, true)

  diags = resp.State.Set(ctx, &plan)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to delete a cmd_local resource.
func (r *resourceCmd) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
  var data resourceCmdData

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  r.init(ctx, data)

  for _, destroy := range data.Destroy {
    cmd := destroy.Cmd
    env := make(map[string]string)
    for k, v := range data.Input {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("INPUT_%s", k)] = s
    }
    stdout, stderr, err := r.shell.Execute(cmd, env)

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
func (r *resourceCmd) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
  tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func (data *resourceCmdData) read_state(ctx context.Context, shell shell, state_only bool) []error {
  var errors []error
  env := make(map[string]string)
  for k, v := range data.Input {
    var s string
    if !v.Null {
      s = v.Value
    }
    env[fmt.Sprintf("INPUT_%s", k)] = s
  }

  for _, reload := range data.Reload {
    name := reload.Name
    cmd := reload.Cmd
    stdout, stderr, err := shell.Execute(cmd, env)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, "cmd", cmd)
    }
    if err == nil {
      if _, found := data.Input[name]; !state_only && found {
        data.Input[name] = types.String{Value: stdout}
      }
      data.State[name] = stdout
    } else {
      errors = append(errors, err)
    }
  }

  return errors
}

// get_update_cmd search for the right command to execute satisfying the update policies of the resource.
func get_update_cmd(state map[string]types.String, plan map[string]types.String, rules resourceCmdData) string {
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
    sort.Strings(triggers)
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
  var plan, state resourceCmdData

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

type updateValidator struct {}

func (_ updateValidator) Description(ctx context.Context) string {
  return "Validates the update policy"
}
func (_ updateValidator) MarkdownDescription(ctx context.Context) string {
  return "Validates the update policy"
}
func (_ updateValidator) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
  var data resourceCmdData

  diags := req.Config.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  type void struct {}
  seen := make(map[string]void)
  conflict := make(map[string]void)

  for _, update0 := range data.Update {
    sort.Strings(update0.Triggers)
    triggers0 := strings.Join(update0.Triggers, ",")
    if _, found := seen[triggers0]; found {
      resp.Diagnostics.AddError("Update ambiguity", fmt.Sprintf("Update rule for %s is duplicated", triggers0))
    }
    seen[triggers0] = void{}

    for _, update1 := range data.Update {
      sort.Strings(update1.Triggers)
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
}
