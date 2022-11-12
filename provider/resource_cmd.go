package provider

import (
  "context"
  "strings"
  "sort"
  "fmt"

  "github.com/hashicorp/terraform-plugin-framework/attr"
  //"github.com/hashicorp/terraform-plugin-framework/datasource"
  "github.com/hashicorp/terraform-plugin-framework/diag"
  //"github.com/hashicorp/terraform-plugin-framework/path"
  //"github.com/hashicorp/terraform-plugin-framework/provider"
  "github.com/hashicorp/terraform-plugin-framework/resource"
  "github.com/hashicorp/terraform-plugin-framework/tfsdk"
  "github.com/hashicorp/terraform-plugin-framework/types"
  "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &cmdResource{}
	_ resource.ResourceWithConfigure   = &cmdResource{}
	_ resource.ResourceWithImportState = &cmdResource{}
)

// cmdResource is a resource handle to a cmd_local resource.
type cmdResource struct {
  shell shell
  shellType string
  shellFactory func(map[string]string) (shell, error)
}

func NewCmdLocalResource() resource.Resource {
  return &cmdResource{
    shell: nil,
    shellType: "local",
    shellFactory: shellLocalFactory,
  }
}

func NewCmdSshResource() resource.Resource {
  return &cmdResource{
    shell: nil,
    shellType: "ssh",
    shellFactory: shellSshFactory,
  }
}

// Metadata returns the data source type name.
func (r *cmdResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
  resp.TypeName = req.ProviderTypeName + "_" + r.shellType
}

// GetSchema returns the Terraform Schema of the cmd_local resource.
func (t *cmdResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
        PlanModifiers: tfsdk.AttributePlanModifiers{
          planModifier2{},
          //resource.UseStateForUnknown(),
        },
        Type: types.MapType{types.StringType},
      },
      "id": {
        Computed:            true,
        MarkdownDescription: "Example identifier",
        PlanModifiers: tfsdk.AttributePlanModifiers{
          resource.UseStateForUnknown(),
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

// Configure adds the provider configured client to the data source.
func (r *cmdResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
}

// cmdResourceModel encodes the data of a cmd_local resource.
type cmdResourceModel struct {
  Id   types.String `tfsdk:"id"`
  Input map[string]types.String `tfsdk:"inputs"`
  State map[string]string `tfsdk:"state"`
  ConnectionOptions map[string]types.String `tfsdk:"connection"`
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

func (r *cmdResource) init(ctx context.Context, data cmdResourceModel) error {
  if r.shell == nil {
    options := make(map[string]string)
    for k, v := range data.ConnectionOptions {
      if v.Null {
        return fmt.Errorf("%s is not known or null", k)
      }
      options[k] = v.Value
    }
    var err error
    r.shell, err = r.shellFactory(options)
    if err != nil {
      return err
    }
  }
  return nil
}

// Create is in charge to crete a cmd_local resource.
func (r *cmdResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
  var data cmdResourceModel

  tflog.Info(ctx, fmt.Sprintf("##### Create:Config #####\n%s\n##### /Create:Config #####", formatVal(req.Config.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### Create:Plan #####\n%s\n##### /Create:Plan #####", formatVal(req.Plan.Raw)))

  diags := req.Config.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  if resp.Diagnostics.HasError() {
    return
  }

  if err := r.init(ctx, data); err != nil {
    resp.Diagnostics.AddError("Connection Error", err.Error())
    return
  }

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
    stdout, stderr, combined, err := r.shell.Execute(cmd, env)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, map[string]any{"cmd": cmd})
    }
    if len(stdout) > 0 {
      tflog.Info(ctx, stdout, map[string]any{"cmd": cmd})
    }

    if err != nil {
      resp.Diagnostics.AddError("Command error", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, combined))
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
func (r *cmdResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
  var data cmdResourceModel

  tflog.Info(ctx, fmt.Sprintf("##### Read:State #####\n%s\n##### /Read:State #####", formatVal(req.State.Raw)))

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  if err := r.init(ctx, data); err != nil {
    resp.Diagnostics.AddError("Connection Error", err.Error())
    return
  }

  data.read_state(ctx, r.shell, true)

  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to update a cmd_local resource.
func (r *cmdResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
  var plan, state cmdResourceModel

  tflog.Info(ctx, fmt.Sprintf("##### Update:Config #####\n%s\n##### /Update:Config #####", formatVal(req.Config.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### Update:State #####\n%s\n##### /Update:State #####", formatVal(req.State.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### Update:Plan #####\n%s\n##### /Update:Plan #####", formatVal(req.Plan.Raw)))

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

  if err := r.init(ctx, plan); err != nil {
    resp.Diagnostics.AddError("Connection Error", err.Error())
    return
  }

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
    stdout, stderr, combined, err := r.shell.Execute(cmd, env)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, map[string]any{"cmd": cmd})
    }
    if len(stdout) > 0 {
      tflog.Info(ctx, stdout, map[string]any{"cmd": cmd})
    }

    if err != nil {
      resp.Diagnostics.AddError("Command error", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, combined))
      return
    }
  }

  plan.read_state(ctx, r.shell, true)

  diags = resp.State.Set(ctx, &plan)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to delete a cmd_local resource.
func (r *cmdResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
  var data cmdResourceModel

  tflog.Info(ctx, fmt.Sprintf("##### Delete:State #####\n%s\n##### /Delete:State #####", formatVal(req.State.Raw)))

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  if err := r.init(ctx, data); err != nil {
    resp.Diagnostics.AddError("Connection Error", err.Error())
    return
  }

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
    stdout, stderr, combined, err := r.shell.Execute(cmd, env)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, map[string]any{"cmd": cmd})
    }
    if len(stdout) > 0 {
      tflog.Info(ctx, stdout, map[string]any{"cmd": cmd})
    }

    if err != nil {
      resp.Diagnostics.AddError("Command error", fmt.Sprintf("Unable to execute command: %s\n%s\n%s", cmd, err, combined))
      return
    }
  }

  resp.State.RemoveResource(ctx)
}

// ImportState is in charge to import a cmd_local resource into terraform.
func (r *cmdResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
  //tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func (data *cmdResourceModel) read_state(ctx context.Context, shell shell, state_only bool) []error {
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
    stdout, stderr, _, err := shell.Execute(cmd, env)

    if len(stderr) > 0 {
      tflog.Warn(ctx, stderr, map[string]any{"cmd": cmd})
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
func get_update_cmd(state map[string]types.String, plan map[string]types.String, rules cmdResourceModel) string {
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
  var plan, state cmdResourceModel

  tflog.Info(ctx, fmt.Sprintf("##### PlanModify:State #####\n%s\n##### /PlanModify:State #####", formatVal(req.State.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### PlanModify:Plan #####\n%s\n##### /PlanModify:Plan #####", formatVal(req.Plan.Raw)))

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

type planModifier2 struct {}

func (_ planModifier2) Description(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ planModifier2) MarkdownDescription(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ planModifier2) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
  tflog.Info(ctx, fmt.Sprintf("##### PlanModify2:State #####\n%s\n##### /PlanModify2:State #####", formatVal(req.State.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### PlanModify2:Plan #####\n%s\n##### /PlanModify2:Plan #####", formatVal(req.Plan.Raw)))

  resp.AttributePlan = types.Map{
    Unknown: false,
    Null: false,
    Elems: map[string]attr.Value{
      "a": types.String{
        Unknown: false,
        Null: false,
        Value: "",
      },
    },
    ElemType: types.StringType,
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
  var data cmdResourceModel

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
