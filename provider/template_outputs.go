package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

type templateOutputsDataSource struct{}

func (*templateOutputsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "coder_template_outputs"
}

func (*templateOutputsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:        "Compatibility data source for the immutable shared resource outputs snapshot supplied to a per-workspace build. Prefer coder_foundation for combined Foundation metadata and outputs. Coder supplies a pinned snapshot; Terraform determines whether required outputs are available.",
		DeprecationMessage: "Use coder_foundation for combined Foundation metadata and outputs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the template, or `import` when unavailable during template import.",
			},
			"outputs": schema.DynamicAttribute{
				Computed:    true,
				Sensitive:   foundationOutputsSensitive(),
				Description: "Object containing shared resource output values. Exact Terraform types and sensitivity are preserved when output metadata is supplied. The entire attribute is sensitive if any root output is sensitive. Defaults to an empty object during template import.",
			},
			"foundation_id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the Foundation supplying the snapshot, or an empty string when unavailable. This metadata does not select a Foundation or revision.",
			},
			"revision_id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the shared resource revision supplying the snapshot, or an empty string when unavailable during template import.",
			},
		},
	}
}

func (*templateOutputsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	outputs, err := foundationOutputValues(ctx, true)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}

	revisionID := helpers.OptionalEnv("CODER_TEMPLATE_RESOURCE_REVISION_ID")
	if revisionID != "" {
		if _, err := uuid.Parse(revisionID); err != nil {
			resp.Diagnostics.AddError("Shared resource revision ID in CODER_TEMPLATE_RESOURCE_REVISION_ID must be a UUID", "")
			return
		}
	}

	foundationID := helpers.OptionalEnv("CODER_FOUNDATION_ID")
	if foundationID != "" {
		if _, err := uuid.Parse(foundationID); err != nil {
			resp.Diagnostics.AddError("Foundation ID in CODER_FOUNDATION_ID must be a UUID", "")
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &struct {
		ID           types.String  `tfsdk:"id"`
		Outputs      types.Dynamic `tfsdk:"outputs"`
		RevisionID   types.String  `tfsdk:"revision_id"`
		FoundationID types.String  `tfsdk:"foundation_id"`
	}{
		ID:           types.StringValue(helpers.OptionalEnvOrDefault("CODER_WORKSPACE_TEMPLATE_ID", templateImportID)),
		Outputs:      outputs,
		RevisionID:   types.StringValue(revisionID),
		FoundationID: types.StringValue(foundationID),
	})...)
}
