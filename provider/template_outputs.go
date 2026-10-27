package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

type templateOutputsDataSource struct{}

func (*templateOutputsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "coder_template_outputs"
}

func (*templateOutputsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Proposed POC data source for the immutable shared resource outputs snapshot supplied to a per-workspace build. Shared resources are provisioned once per template; per-workspace resources are provisioned for each workspace. Coder excludes sensitive outputs before supplying the snapshot. Build gating is enforced by the server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the template, or `import` when unavailable during template import.",
			},
			"outputs": schema.DynamicAttribute{
				Computed:    true,
				Description: "Object containing non-sensitive shared resource output values, preserving nested objects, arrays, numbers, booleans, and nulls. Access values directly, for example `data.coder_template_outputs.current.outputs.namespace`. Defaults to an empty object during template import.",
			},
			"revision_id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the shared resource revision supplying the snapshot, or an empty string when unavailable during template import.",
			},
		},
	}
}

func (*templateOutputsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	outputsJSON := helpers.OptionalEnvOrDefault("CODER_TEMPLATE_OUTPUTS", "{}")
	// Use Terraform's JSON type inference, not Go's float64 decoding or SDK
	// maps, to preserve heterogeneous tuples, nulls, and number precision.
	impliedType, err := ctyjson.ImpliedType([]byte(outputsJSON))
	if err != nil || !impliedType.IsObjectType() {
		resp.Diagnostics.AddError("Shared resource outputs in CODER_TEMPLATE_OUTPUTS must be a JSON object", "")
		return
	}
	typeJSON, err := ctyjson.MarshalType(impliedType)
	if err != nil {
		resp.Diagnostics.AddError("Unable to determine shared resource output types", "")
		return
	}
	tfType, err := tftypes.ParseJSONType(typeJSON)
	if err != nil {
		resp.Diagnostics.AddError("Unable to determine shared resource output types", "")
		return
	}
	tfValue, err := tftypes.ValueFromJSON([]byte(outputsJSON), tfType)
	if err != nil {
		// Decoder errors can include output values. Never expose the payload.
		resp.Diagnostics.AddError("Shared resource outputs in CODER_TEMPLATE_OUTPUTS must be a JSON object", "")
		return
	}
	outputs, err := types.DynamicType.ValueFromTerraform(ctx, tfValue)
	if err != nil {
		resp.Diagnostics.AddError("Unable to convert shared resource output values", "")
		return
	}

	revisionID := helpers.OptionalEnv("CODER_TEMPLATE_RESOURCE_REVISION_ID")
	if revisionID != "" {
		if _, err := uuid.Parse(revisionID); err != nil {
			resp.Diagnostics.AddError("Shared resource revision ID in CODER_TEMPLATE_RESOURCE_REVISION_ID must be a UUID", "")
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &struct {
		ID         types.String  `tfsdk:"id"`
		Outputs    types.Dynamic `tfsdk:"outputs"`
		RevisionID types.String  `tfsdk:"revision_id"`
	}{
		ID:         types.StringValue(helpers.OptionalEnvOrDefault("CODER_WORKSPACE_TEMPLATE_ID", templateImportID)),
		Outputs:    outputs.(types.Dynamic),
		RevisionID: types.StringValue(revisionID),
	})...)
}
