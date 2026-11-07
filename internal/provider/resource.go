package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Adatage/terraform-provider-adatage/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const operationTimeout = 45 * time.Minute

var errGone = errors.New("not found")

func gone(err error) error {
	if client.IsNotFound(err) {
		return errGone
	}
	return err
}

type Resource struct {
	Name         string
	Desc         string
	Attrs        []Attr
	ImportFields []string
	ImportParse  func(id string) (Values, error)

	Create func(ctx context.Context, c *client.Client, plan Values) (Values, error)
	Read   func(ctx context.Context, c *client.Client, state Values) (map[string]any, error)
	Update func(ctx context.Context, c *client.Client, plan, state Values) error
	Delete func(ctx context.Context, c *client.Client, state Values) error
}

func (r *Resource) importFields() []string {
	if len(r.ImportFields) > 0 {
		return r.ImportFields
	}
	return []string{"id"}
}

func mods[T any](a Attr, replace, replaceIfConfigured, useState T) []T {
	var out []T
	if a.ForceNew {
		if a.Computed && !a.Required {
			out = append(out, replaceIfConfigured)
		} else {
			out = append(out, replace)
		}
	}
	if a.Computed && a.Stable && !a.ForceNew {
		out = append(out, useState)
	}
	return out
}

func resourceAttr(a Attr) schema.Attribute {
	desc := a.Desc
	switch a.Kind {
	case Int:
		return schema.Int64Attribute{Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
			PlanModifiers: mods[planmodifier.Int64](a, int64planmodifier.RequiresReplace(), int64planmodifier.RequiresReplaceIfConfigured(), int64planmodifier.UseStateForUnknown())}
	case Bool:
		return schema.BoolAttribute{Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
			PlanModifiers: mods[planmodifier.Bool](a, boolplanmodifier.RequiresReplace(), boolplanmodifier.RequiresReplaceIfConfigured(), boolplanmodifier.UseStateForUnknown())}
	case StringList:
		return schema.ListAttribute{ElementType: types.StringType, Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
			PlanModifiers: mods[planmodifier.List](a, listplanmodifier.RequiresReplace(), listplanmodifier.RequiresReplaceIfConfigured(), listplanmodifier.UseStateForUnknown())}
	case StringMap:
		return schema.MapAttribute{ElementType: types.StringType, Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
			PlanModifiers: mods[planmodifier.Map](a, mapplanmodifier.RequiresReplace(), mapplanmodifier.RequiresReplaceIfConfigured(), mapplanmodifier.UseStateForUnknown())}
	case List:
		return schema.ListNestedAttribute{Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
			NestedObject:  schema.NestedAttributeObject{Attributes: resourceAttrs(a.Nested)},
			PlanModifiers: mods[planmodifier.List](a, listplanmodifier.RequiresReplace(), listplanmodifier.RequiresReplaceIfConfigured(), listplanmodifier.UseStateForUnknown())}
	case Object:
		return schema.SingleNestedAttribute{Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
			Attributes:    resourceAttrs(a.Nested),
			PlanModifiers: mods[planmodifier.Object](a, objectplanmodifier.RequiresReplace(), objectplanmodifier.RequiresReplaceIfConfigured(), objectplanmodifier.UseStateForUnknown())}
	}
	return schema.StringAttribute{Description: desc, Required: a.Required, Optional: a.Optional, Computed: a.Computed, Sensitive: a.Sensitive,
		PlanModifiers: mods[planmodifier.String](a, stringplanmodifier.RequiresReplace(), stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown())}
}

func resourceAttrs(attrs []Attr) map[string]schema.Attribute {
	out := map[string]schema.Attribute{}
	for _, a := range attrs {
		out[a.Name] = resourceAttr(a)
	}
	return out
}

func dataAttr(a Attr) dsschema.Attribute {
	desc, req, opt, comp, sens := a.Desc, a.Required, a.Optional, a.Computed, a.Sensitive
	switch a.Kind {
	case Int:
		return dsschema.Int64Attribute{Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens}
	case Bool:
		return dsschema.BoolAttribute{Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens}
	case StringList:
		return dsschema.ListAttribute{ElementType: types.StringType, Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens}
	case StringMap:
		return dsschema.MapAttribute{ElementType: types.StringType, Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens}
	case List:
		return dsschema.ListNestedAttribute{Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens, NestedObject: dsschema.NestedAttributeObject{Attributes: dataAttrs(a.Nested)}}
	case Object:
		return dsschema.SingleNestedAttribute{Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens, Attributes: dataAttrs(a.Nested)}
	}
	return dsschema.StringAttribute{Description: desc, Required: req, Optional: opt, Computed: comp, Sensitive: sens}
}

func dataAttrs(attrs []Attr) map[string]dsschema.Attribute {
	out := map[string]dsschema.Attribute{}
	for _, a := range attrs {
		out[a.Name] = dataAttr(a)
	}
	return out
}

func decode(attrs []Attr, raw tftypes.Value) (Values, error) {
	x, err := toGo(raw)
	if err != nil {
		return nil, err
	}
	m, _ := x.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	return Values(m), nil
}

func encode(attrs []Attr, v Values) (tftypes.Value, error) {
	clean := map[string]any{}
	for k, x := range v {
		if !isUnknown(x) {
			clean[k] = x
		}
	}
	return fromGo(objectType(attrs), clean)
}

type genericResource struct {
	spec *Resource
	c    *client.Client
}

var (
	_ resource.Resource                = (*genericResource)(nil)
	_ resource.ResourceWithConfigure   = (*genericResource)(nil)
	_ resource.ResourceWithImportState = (*genericResource)(nil)
)

func (r *genericResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.Name
}

func (r *genericResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: r.spec.Desc, Attributes: resourceAttrs(r.spec.Attrs)}
}

func (r *genericResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("%T", req.ProviderData))
		return
	}
	r.c = c
}

func (r *genericResource) fail(d *diag.Diagnostics, what string, err error) {
	d.AddError(fmt.Sprintf("Could not %s %s", what, r.spec.Name), err.Error())
}

func (r *genericResource) save(state *tfsdk.State, d *diag.Diagnostics, v Values) {
	raw, err := encode(r.spec.Attrs, v)
	if err != nil {
		d.AddError("Could not build the state", err.Error())
		return
	}
	state.Raw = raw
}

func (r *genericResource) refresh(ctx context.Context, prior Values) (Values, error) {
	obj, err := r.spec.Read(ctx, r.c, prior)
	if err != nil {
		return nil, err
	}
	out := readObject(r.spec.Attrs, obj, prior)
	// Keep what the plan said for attributes the API leaves out or spells differently, so that Terraform sees no
	// change.
	return out, nil
}

func (r *genericResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	plan, err := decode(r.spec.Attrs, req.Plan.Raw)
	if err != nil {
		r.fail(&resp.Diagnostics, "read the plan of", err)
		return
	}
	ids, err := r.spec.Create(ctx, r.c, plan)
	if err != nil {
		r.fail(&resp.Diagnostics, "create", err)
		return
	}
	for k, v := range ids {
		plan[k] = v
	}
	state, err := r.refresh(ctx, plan)
	if err != nil {
		r.save(&resp.State, &resp.Diagnostics, mergeKnown(plan, ids))
		r.fail(&resp.Diagnostics, "read the new", err)
		return
	}
	r.save(&resp.State, &resp.Diagnostics, state)
}

func mergeKnown(plan, ids Values) Values {
	out := Values{}
	for k, v := range plan {
		if !isUnknown(v) {
			out[k] = v
		}
	}
	for k, v := range ids {
		out[k] = v
	}
	return out
}

func (r *genericResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	prior, err := decode(r.spec.Attrs, req.State.Raw)
	if err != nil {
		r.fail(&resp.Diagnostics, "read the state of", err)
		return
	}
	state, err := r.refresh(ctx, prior)
	if errors.Is(err, errGone) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		r.fail(&resp.Diagnostics, "read", err)
		return
	}
	r.save(&resp.State, &resp.Diagnostics, state)
}

func (r *genericResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	plan, err := decode(r.spec.Attrs, req.Plan.Raw)
	if err != nil {
		r.fail(&resp.Diagnostics, "read the plan of", err)
		return
	}
	prior, err := decode(r.spec.Attrs, req.State.Raw)
	if err != nil {
		r.fail(&resp.Diagnostics, "read the state of", err)
		return
	}
	if r.spec.Update != nil {
		if err := r.spec.Update(ctx, r.c, plan, prior); err != nil {
			r.fail(&resp.Diagnostics, "update", err)
			return
		}
	}
	for k, v := range plan {
		if isUnknown(v) {
			plan[k] = prior[k]
		}
	}
	state, err := r.refresh(ctx, plan)
	if err != nil {
		r.fail(&resp.Diagnostics, "read the updated", err)
		return
	}
	r.save(&resp.State, &resp.Diagnostics, state)
}

func (r *genericResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	prior, err := decode(r.spec.Attrs, req.State.Raw)
	if err != nil {
		r.fail(&resp.Diagnostics, "read the state of", err)
		return
	}
	if err := r.spec.Delete(ctx, r.c, prior); err != nil && !errors.Is(err, errGone) {
		r.fail(&resp.Diagnostics, "delete", err)
	}
}

func (r *genericResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	fields := r.spec.importFields()
	prior := Values{}
	switch {
	case r.spec.ImportParse != nil:
		var err error
		if prior, err = r.spec.ImportParse(req.ID); err != nil {
			resp.Diagnostics.AddError("Invalid import ID", err.Error())
			return
		}
	case len(fields) == 1:
		prior[fields[0]] = req.ID
	default:
		parts := strings.Split(req.ID, "/")
		if len(parts) != len(fields) {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected %s", strings.Join(fields, "/")))
			return
		}
		for i, f := range fields {
			prior[f] = parts[i]
		}
	}
	state, err := r.refresh(ctx, prior)
	if errors.Is(err, errGone) {
		resp.Diagnostics.AddError("Not found", "no such "+r.spec.Name)
		return
	}
	if err != nil {
		r.fail(&resp.Diagnostics, "import", err)
		return
	}
	r.save(&resp.State, &resp.Diagnostics, state)
}

type DataSource struct {
	Name  string
	Desc  string
	Attrs []Attr
	Read  func(ctx context.Context, c *client.Client, cfg Values) (map[string]any, error)
}

type genericData struct {
	spec *DataSource
	c    *client.Client
}

var (
	_ datasource.DataSource              = (*genericData)(nil)
	_ datasource.DataSourceWithConfigure = (*genericData)(nil)
)

func (d *genericData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.spec.Name
}

func (d *genericData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{Description: d.spec.Desc, Attributes: dataAttrs(d.spec.Attrs)}
}

func (d *genericData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c, ok := req.ProviderData.(*client.Client); ok {
		d.c = c
	}
}

func (d *genericData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	cfg, err := decode(d.spec.Attrs, req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Could not read the configuration", err.Error())
		return
	}
	obj, err := d.spec.Read(ctx, d.c, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Could not read "+d.spec.Name, err.Error())
		return
	}
	raw, err := encode(d.spec.Attrs, readObject(d.spec.Attrs, obj, cfg))
	if err != nil {
		resp.Diagnostics.AddError("Could not build the state", err.Error())
		return
	}
	resp.State.Raw = raw
}
