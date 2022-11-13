package cmd

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
  "github.com/hashicorp/terraform-plugin-framework/path"
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
          inputPlanModifier{},
        },
        Type: types.MapType{types.StringType},
      },
      "state": {
        Computed:            true,
        MarkdownDescription: "State",
        PlanModifiers: tfsdk.AttributePlanModifiers{
          statePlanModifier{},
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
            Type:                types.SetType{types.StringType},
          },
          "invalidates": {
            MarkdownDescription: "What state varaibles are invalidated",
            Optional:            true,
            Type:                types.SetType{types.StringType},
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
  State map[string]types.String `tfsdk:"state"`
  ConnectionOptions map[string]types.String `tfsdk:"connection"`
  Reload []cmdResourceReloadModel `tfsdk:"reload"`
  Update []cmdResourceUpdateModel `tfsdk:"update"`
  Create []cmdResourceCreateModel `tfsdk:"create"`
  Destroy []cmdResourceDestroyModel `tfsdk:"destroy"`
}

type cmdResourceReloadModel struct {
  Name string `tfsdk:"name"`
  Cmd string `tfsdk:"cmd"`
}
type cmdResourceUpdateModel struct {
  Triggers []string `tfsdk:"triggers"`
  Invalidates []string `tfsdk:"invalidates"`
  Cmd string `tfsdk:"cmd"`
}
type cmdResourceCreateModel struct {
  Cmd string `tfsdk:"cmd"`
}
type cmdResourceDestroyModel struct {
  Cmd string `tfsdk:"cmd"`
}

//type cmdResourceData struct {
//  Id string
//  Input map[string]string
//  State map[string]string
//  ConnectionOptions map[string]string
//  Reload map[string]string
//  Update []cmdResourceUpdateData
//  CreateCmd string
//  DestroyCmd string
//}
//
//type cmdResourceUpdateData struct {
//  Triggers []string
//  Invalidates []string
//  Cmd string
//}
//
//func tryString(str types.String) string {
//  if str.IsUnknown() || str.IsNull() {
//    return ""
//  }
//  return str.Value
//}
//
//func (model *cmdResourceModel) toData() cmdResourceData {
//  data := cmdResourceData{
//    Id: "",
//    Input: make(map[string]string),
//    State: make(map[string]string),
//    ConnectionOptions: make(map[string]string),
//    Reload: make(map[string]string),
//    Update: []cmdResourceUpdateData{},
//    CreateCmd: "",
//    DestroyCmd: "",
//  }
//
//  if model == nil {
//    return data
//  }
//
//  data.Id = tryString(model.Id)
//  for k, v := range model.Input {
//    data.Input[k] = tryString(v)
//  }
//  for k, v := range model.State {
//    data.State[k] = tryString(v)
//  }
//  for k, v := range model.ConnectionOptions {
//    data.ConnectionOptions[k] = tryString(v)
//  }
//  for _, reloadModel := range model.Reload {
//    data.Reload[reloadModel.Name] = reloadModel.Cmd
//  }
//  for _, updateModel := range model.Update {
//    updateData := cmdResourceUpdateData{
//      Triggers: updateModel.Triggers,
//      Invalidates: updateModel.Invalidates,
//      Cmd: updateModel.Cmd,
//    }
//    sort.Strings(updateData.Triggers)
//    sort.Strings(updateData.Invalidates)
//    data.Update = append(data.Update, updateData)
//  }
//  for _, createModel := range model.Create {
//    data.CreateCmd = createModel.Cmd
//  }
//  for _, destroyModel := range model.Destroy {
//    data.DestroyCmd = destroyModel.Cmd
//  }
//
//  return data
//}


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

  data.State = make(map[string]types.String)
  data.read_state(ctx, r.shell, true)

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
    plan.State = make(map[string]types.String)
  }

  if resp.Diagnostics.HasError() {
    return
  }

  if err := r.init(ctx, plan); err != nil {
    resp.Diagnostics.AddError("Connection Error", err.Error())
    return
  }

  update := get_update(state.Input, plan.Input, state)

  if update != nil {
    cmd := update.Cmd
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
    for k, v := range state.State {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("STATE_%s", k)] = s
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
    for k, v := range data.State {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("STATE_%s", k)] = s
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
    for k, v := range data.State {
      var s string
      if !v.Null {
        s = v.Value
      }
      env[fmt.Sprintf("STATE_%s", k)] = s
    }
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
        data.Input[name] = types.StringValue(stdout)
      }
      data.State[name] = types.StringValue(stdout)
    } else {
      errors = append(errors, err)
    }
  }

  return errors
}

// get_update search for the right command to execute satisfying the update policies of the resource.
func get_update(state map[string]types.String, plan map[string]types.String, rules cmdResourceModel) *cmdResourceUpdateModel {
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
  if len(modified) == 0 {
    return nil
  }
  sort.Strings(modified)

  var trig []string
  var rule *cmdResourceUpdateModel = nil

  for i := range rules.Update {
    update := &rules.Update[i]
    triggers := update.Triggers
    sort.Strings(triggers)
    if len(triggers) == 0 {
      if rule == nil {
        rule = update
      }
    } else if len(triggers) >= len(modified) && (len(trig) == 0 || len(trig) > len(triggers)) {
      _, _, right := sorted_list_3way(triggers, modified)
      if len(right) == 0 {
        trig = triggers
        rule = update
      }
    }
  }

  return rule
}

type inputPlanModifier struct {}

func (_ inputPlanModifier) Description(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ inputPlanModifier) MarkdownDescription(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ inputPlanModifier) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
  var plan, state cmdResourceModel

  tflog.Info(ctx, fmt.Sprintf("##### InputPlanModify:State #####\n%s\n##### /InputPlanModify:State #####", formatVal(req.State.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### InputPlanModify:Plan #####\n%s\n##### /InputPlanModify:Plan #####", formatVal(req.Plan.Raw)))

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

  rule := get_update(state.Input, plan.Input, state)
  if rule == nil {
    resp.RequiresReplace = true
  }
}

type statePlanModifier struct {}

func (_ statePlanModifier) Description(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ statePlanModifier) MarkdownDescription(ctx context.Context) string {
  return "Checks if the resource must be replaced depending on which inputs are plan to be modified"
}

func (_ statePlanModifier) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
  tflog.Info(ctx, fmt.Sprintf("##### StatePlanModify:State #####\n%s\n##### /StatePlanModify:State #####", formatVal(req.State.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### StatePlanModify:Config #####\n%s\n##### /StatePlanModify:Config #####", formatVal(req.Config.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### StatePlanModify:Plan #####\n%s\n##### /StatePlanModify:Plan #####", formatVal(req.Plan.Raw)))

  // No modification on destroy
  if req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() {
    return
  }

  tflog.Info(ctx, "##### Apply StatePlanModify #####")

  var config cmdResourceModel

  resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

  configReloadData := config.Reload
  stateData := map[string]types.String{}
  stateInputData := map[string]types.String{}
  stateReloadData := []cmdResourceReloadModel{}
  planInputData := map[string]types.String{}

  resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("inputs"), &planInputData)...)

  //configReloadData := config.Reload
  //planInputData := plan.Input

  // If this is not a creation, we must read the state
  if !req.State.Raw.IsNull() && req.State.Raw.IsKnown() {
    resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("state"), &stateData)...)
    resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("inputs"), &stateInputData)...)
    resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("reload"), &stateReloadData)...)
  }


  type void struct{}
  stateReload := make(map[string]string)
  elems := make(map[string]attr.Value)

  rule := get_update(stateInputData, planInputData, config)
  invalidatesAll := rule != nil && rule.Invalidates == nil

  for _, reload := range stateReloadData {
    stateReload[reload.Name] = reload.Cmd
  }
  for _, reload := range configReloadData {
    name := reload.Name
    value, valueFound := stateData[name]
    stateCmd, cmdFound := stateReload[name]
    if invalidatesAll || !valueFound || !cmdFound || stateCmd != reload.Cmd {
      elems[name] = types.StringUnknown()
    } else {
      elems[name] = value
    }
  }

  if rule != nil {
    for _, invalidate := range rule.Invalidates {
      elems[invalidate] = types.StringUnknown()
    }
  }

  resp.AttributePlan = types.Map{
    Unknown: false,
    Null: false,
    Elems: elems,
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
