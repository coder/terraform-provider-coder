package provider

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

type foundationDataSource struct{}

func (*foundationDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "coder_foundation"
}

func (*foundationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Metadata and the immutable outputs snapshot supplied by Coder for the current Foundation. In Foundation runs this is the Foundation itself; in workspace builds this is the attached Foundation. No Foundation selection or API calls are performed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the Foundation, or `import` when no Foundation metadata is available. Never falls back to the template ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the Foundation, or `default` when unavailable.",
			},
			"template_ids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "IDs of templates attached to the Foundation, or an empty list when unbound or unavailable.",
			},
			"snapshot_id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the Foundation run that produced the supplied outputs snapshot, or an empty string when unavailable. This metadata does not select a run.",
			},
			"outputs": schema.DynamicAttribute{
				Computed:    true,
				Sensitive:   false,
				Description: "Object containing only non-sensitive Foundation outputs with their exact Terraform types. Can be used in for_each without nonsensitive. Defaults to an empty object when no non-sensitive outputs are supplied; use `try` for optional outputs.",
			},
			"sensitive_outputs": schema.DynamicAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Sensitive object containing only sensitive Foundation outputs with their exact Terraform types, from the same pinned snapshot as outputs. Always marked sensitive, including when empty. Defaults to an empty object when no sensitive outputs are supplied; use `try` for optional outputs.",
			},
		},
	}
}

func (*foundationDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	id := helpers.OptionalEnv("CODER_FOUNDATION_ID")
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			resp.Diagnostics.AddError("Foundation ID in CODER_FOUNDATION_ID must be a UUID", "")
			return
		}
	}
	snapshotID := helpers.OptionalEnv("CODER_FOUNDATION_SNAPSHOT_ID")
	if snapshotID != "" {
		if _, err := uuid.Parse(snapshotID); err != nil {
			resp.Diagnostics.AddError("Foundation snapshot ID in CODER_FOUNDATION_SNAPSHOT_ID must be a UUID", "")
			return
		}
	}
	var templateIDs []string
	if err := json.Unmarshal([]byte(helpers.OptionalEnvOrDefault("CODER_FOUNDATION_TEMPLATE_IDS", "[]")), &templateIDs); err != nil || templateIDs == nil {
		resp.Diagnostics.AddError("Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs", "")
		return
	}
	for _, templateID := range templateIDs {
		if _, err := uuid.Parse(templateID); err != nil {
			resp.Diagnostics.AddError("Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs", "")
			return
		}
	}
	templates, diags := types.ListValueFrom(ctx, types.StringType, templateIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	outputs, sensitiveOutputs, err := splitFoundationOutputValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	if id == "" {
		id = templateImportID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &struct {
		ID               types.String  `tfsdk:"id"`
		Name             types.String  `tfsdk:"name"`
		TemplateIDs      types.List    `tfsdk:"template_ids"`
		SnapshotID       types.String  `tfsdk:"snapshot_id"`
		Outputs          types.Dynamic `tfsdk:"outputs"`
		SensitiveOutputs types.Dynamic `tfsdk:"sensitive_outputs"`
	}{
		ID:               types.StringValue(id),
		Name:             types.StringValue(helpers.OptionalEnvOrDefault("CODER_FOUNDATION_NAME", "default")),
		TemplateIDs:      templates,
		SnapshotID:       types.StringValue(snapshotID),
		Outputs:          outputs,
		SensitiveOutputs: sensitiveOutputs,
	})...)
}
