package cmd

import (
  "context"
  "strings"
  "sort"
  "fmt"
  "regexp"

  "github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
  "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &resourceCommand{}
	_ resource.ResourceWithConfigure   = &resourceCommand{}
	_ resource.ResourceWithImportState = &resourceCommand{}
)

// resourceCommand is a resource handle to a cmd_local resource.
type resourceCommand struct {
  shell shell
  shellFactory shellFactory
}

// Metadata returns the data source type name.
func (r *resourceCommand) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
  resp.TypeName = req.ProviderTypeName + "_" + r.shellFactory.Name
}

// GetSchema returns the Terraform Schema of the cmd_local resource.
func (r *resourceCommand) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
  return tfsdk.Schema{
    Description: "Custom resource managed by local shell scripts",
    MarkdownDescription: "Custom resource managed by local shell scripts",

    Attributes: map[string]tfsdk.Attribute{
      "connection": {
        Optional:            true,
        MarkdownDescription: "Connection Options",
        Attributes: tfsdk.SingleNestedAttributes(r.shellFactory.Schema),
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
            Validators: []tfsdk.AttributeValidator{
              setvalidator.SizeAtLeast(1),
              setvalidator.ValuesAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z]\w*$`), "must start with a letter and contain only letters, digits and underscore")),
            },
          },
          "reloads": {
            MarkdownDescription: "What state variables must be reloaded",
            Optional:            true,
            Type:                types.SetType{types.StringType},
            Validators: []tfsdk.AttributeValidator{
              updateReloadValidator{},
            },
          },
          "cmd": {
            MarkdownDescription: "Command to execute",
            Required:            true,
            Type:                types.StringType,
          },
        },
        Validators: []tfsdk.AttributeValidator{
          updateAmbiguityValidator{},
        },
      },
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
func (r *resourceCommand) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
}

// resourceCommandModel encodes the data of a cmd_local resource.
type resourceCommandModel struct {
  Id   types.String `tfsdk:"id"`
  Input map[string]types.String `tfsdk:"inputs"`
  State map[string]types.String `tfsdk:"state"`
  ConnectionOptions types.Object `tfsdk:"connection"`
  Read []resourceCommandReadModel `tfsdk:"read"`
  Update []resourceCommandUpdateModel `tfsdk:"update"`
  Create []resourceCommandCreateModel `tfsdk:"create"`
  Destroy []resourceCommandDestroyModel `tfsdk:"destroy"`
}

type resourceCommandReadModel struct {
  Name string `tfsdk:"name"`
  Cmd string `tfsdk:"cmd"`
}
type resourceCommandUpdateModel struct {
  Triggers []string `tfsdk:"triggers"`
  Reloads []string `tfsdk:"reloads"`
  Cmd string `tfsdk:"cmd"`
}
type resourceCommandCreateModel struct {
  Cmd string `tfsdk:"cmd"`
}
type resourceCommandDestroyModel struct {
  Cmd string `tfsdk:"cmd"`
}

//type resourceCommandData struct {
//  Id string
//  Input map[string]string
//  State map[string]string
//  ConnectionOptions map[string]string
//  Read map[string]string
//  Update []resourceCommandUpdateData
//  CreateCmd string
//  DestroyCmd string
//}
//
//type resourceCommandUpdateData struct {
//  Triggers []string
//  Reloads []string
//  Cmd string
//}
//
//
//func (model *resourceCommandModel) toData() resourceCommandData {
//  data := resourceCommandData{
//    Id: "",
//    Input: make(map[string]string),
//    State: make(map[string]string),
//    ConnectionOptions: make(map[string]string),
//    Read: make(map[string]string),
//    Update: []resourceCommandUpdateData{},
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
//  for _, reloadModel := range model.Read {
//    data.Read[reloadModel.Name] = readModel.Cmd
//  }
//  for _, updateModel := range model.Update {
//    updateData := resourceCommandUpdateData{
//      Triggers: updateModel.Triggers,
//      Reloads: updateModel.Reloads,
//      Cmd: updateModel.Cmd,
//    }
//    sort.Strings(updateData.Triggers)
//    sort.Strings(updateData.Reloads)
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

func tryString(str types.String) string {
  if str.IsUnknown() || str.IsNull() {
    return ""
  }
  return str.Value
}

func (r *resourceCommand) init(ctx context.Context, data resourceCommandModel) diag.Diagnostics {
  var diags diag.Diagnostics
  if r.shell == nil {
    r.shell, diags = r.shellFactory.Create(ctx, data.ConnectionOptions)
    if len(diags) > 0 {
      return diags
    }
  }
  return nil
}

// Create is in charge to crete a cmd_local resource.
func (r *resourceCommand) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
  var data resourceCommandModel

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


  if d := r.init(ctx, data); len(d) > 0 {
    resp.Diagnostics.Append(d...)
    return
  }

  for _, create := range data.Create {
    cmd := create.Cmd
    env := make(map[string]string)
    for k, v := range data.Input {
      env[fmt.Sprintf("INPUT_%s", k)] = tryString(v)
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
  data.readState(ctx, r.shell, nil, true)

  data.Id = types.StringValue(generate_id())

  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to read the state of a cmd_local resource during a refresh.
func (r *resourceCommand) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
  var data resourceCommandModel

  tflog.Info(ctx, fmt.Sprintf("##### Read:State #####\n%s\n##### /Read:State #####", formatVal(req.State.Raw)))

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  if d := r.init(ctx, data); len(d) > 0 {
    resp.Diagnostics.Append(d...)
    return
  }

  data.readState(ctx, r.shell, nil, true)

  diags = resp.State.Set(ctx, &data)
  resp.Diagnostics.Append(diags...)
}

// Read is in charge to update a cmd_local resource.
func (r *resourceCommand) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
  var plan, state resourceCommandModel

  tflog.Info(ctx, fmt.Sprintf("##### Update:Config #####\n%s\n##### /Update:Config #####", formatVal(req.Config.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### Update:State #####\n%s\n##### /Update:State #####", formatVal(req.State.Raw)))
  tflog.Info(ctx, fmt.Sprintf("##### Update:Plan #####\n%s\n##### /Update:Plan #####", formatVal(req.Plan.Raw)))

  resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
  resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

  plan.Id = state.Id

  if resp.Diagnostics.HasError() {
    return
  }

  if d := r.init(ctx, plan); len(d) > 0 {
    resp.Diagnostics.Append(d...)
    return
  }

  update := plan.get_update(state.Input, plan.Input)

  if update != nil {
    cmd := update.Cmd
    env := make(map[string]string)

    for k, v := range plan.Input {
      env[fmt.Sprintf("INPUT_%s", k)] = tryString(v)
    }
    for k, v := range state.Input {
      env[fmt.Sprintf("PREVIOUS_%s", k)] = tryString(v)
    }
    for k, v := range state.State {
      env[fmt.Sprintf("STATE_%s", k)] = tryString(v)
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

  var reloads []string
  for name, value := range plan.State {
    if value.IsUnknown() {
      reloads = append(reloads, name)
    }
  }
  plan.readState(ctx, r.shell, reloads, true)


  resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
  tflog.Info(ctx, fmt.Sprintf("##### Update:Output #####\n%s\n##### /Update:Output #####", formatVal(resp.State.Raw)))
}

// Read is in charge to delete a cmd_local resource.
func (r *resourceCommand) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
  var data resourceCommandModel

  tflog.Info(ctx, fmt.Sprintf("##### Delete:State #####\n%s\n##### /Delete:State #####", formatVal(req.State.Raw)))

  diags := req.State.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  if resp.Diagnostics.HasError() {
    return
  }

  if d := r.init(ctx, data); len(d) > 0 {
    resp.Diagnostics.Append(d...)
    return
  }

  for _, destroy := range data.Destroy {
    cmd := destroy.Cmd
    env := make(map[string]string)
    for k, v := range data.Input {
      env[fmt.Sprintf("INPUT_%s", k)] = tryString(v)
    }
    for k, v := range data.State {
      env[fmt.Sprintf("STATE_%s", k)] = tryString(v)
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
func (r *resourceCommand) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
  //tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func (data *resourceCommandModel) readState(ctx context.Context, shell shell, variables []string, state_only bool) []error {
  var errors []error

  type void struct{}
  varShouldBeRead := make(map[string]void)
  if variables == nil {
    for _, read := range data.Read {
      varShouldBeRead[read.Name] = void{}
    }
  } else {
    for _, v := range variables {
      varShouldBeRead[v] = void{}
    }
  }
  env := make(map[string]string)
  for k, v := range data.Input {
    env[fmt.Sprintf("INPUT_%s", k)] = tryString(v)
  }
  for k, v := range data.State {
    env[fmt.Sprintf("STATE_%s", k)] = tryString(v)
  }

  for _, read := range data.Read {
    name := read.Name
    cmd := read.Cmd

    if _, found := varShouldBeRead[name]; !found {
      continue
    }
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
func (rules *resourceCommandModel) get_update(state map[string]types.String, plan map[string]types.String) *resourceCommandUpdateModel {
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
  var rule *resourceCommandUpdateModel = nil

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
  var plan, state resourceCommandModel

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

  rule := state.get_update(state.Input, plan.Input)
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

  var config resourceCommandModel

  resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

  configReadData := config.Read
  stateData := map[string]types.String{}
  stateInputData := map[string]types.String{}
  stateReadData := []resourceCommandReadModel{}
  planInputData := map[string]types.String{}

  resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("inputs"), &planInputData)...)

  // If this is not a resource creation, we must read the state
  if !req.State.Raw.IsNull() && req.State.Raw.IsKnown() {
    resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("state"), &stateData)...)
    resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("inputs"), &stateInputData)...)
    resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("read"), &stateReadData)...)
  }

  stateRead := make(map[string]string)
  elems := make(map[string]attr.Value)

  rule := config.get_update(stateInputData, planInputData)
  reloadAll := rule != nil && rule.Reloads == nil

  for _, read := range stateReadData {
    stateRead[read.Name] = read.Cmd
  }
  for _, read := range configReadData {
    name := read.Name
    elem := types.StringUnknown()
    if !reloadAll {
      stateCmd, cmdFound := stateRead[name]
      if cmdFound && stateCmd == read.Cmd {
        value, valueFound := stateData[name]
        if valueFound {
          elem = value
        }
      }
    }
    elems[name] = elem
  }

  if rule != nil {
    for _, reload := range rule.Reloads {
      elems[reload] = types.StringUnknown()
    }
  }

  resp.AttributePlan = types.Map{
    Unknown: false,
    Null: false,
    Elems: elems,
    ElemType: types.StringType,
  }
}

type updateReloadValidator struct {}

func (_ updateReloadValidator) Description(ctx context.Context) string {
  return "Validates the update policy"
}
func (_ updateReloadValidator) MarkdownDescription(ctx context.Context) string {
  return "Validates the update policy"
}
func (_ updateReloadValidator) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
  if req.AttributeConfig.IsUnknown() || req.AttributeConfig.IsNull() {
    return
  }

  var readModel []resourceCommandReadModel

  diags := req.Config.GetAttribute(ctx, path.Root("read"), &readModel)
  resp.Diagnostics.Append(diags...)
  if diags.HasError() {
    return
  }

  type void struct{}
  vars := make(map[string]void)

  for _, read := range readModel {
    vars[read.Name] = void{}
  }

  var reloads []types.String
  diags = tfsdk.ValueAs(ctx, req.AttributeConfig, &reloads)

  for _, name := range reloads {
    if !name.IsUnknown() && !name.IsNull() {
      if _, found := vars[name.ValueString()]; !found {
        path := req.AttributePath.AtSetValue(name)
        resp.Diagnostics.AddAttributeError(path, "Invalid reload specification for update block", fmt.Sprintf("%s request the reloading of the variable %s, but it does not exit (ie: there is no read block with such a name).", path, name))
      }
    }
  }
}


type updateAmbiguityValidator struct {}

func (_ updateAmbiguityValidator) Description(ctx context.Context) string {
  return "Validates the update policy"
}
func (_ updateAmbiguityValidator) MarkdownDescription(ctx context.Context) string {
  return "Validates the update policy"
}
func (_ updateAmbiguityValidator) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
  var data resourceCommandModel

  diags := req.Config.Get(ctx, &data)
  resp.Diagnostics.Append(diags...)

  // `triggers` of all update blocks must be ordered for the following to work
  for _, update := range data.Update {
    sort.Strings(update.Triggers)
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
}
