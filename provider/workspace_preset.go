package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/mapstructure"
)

type WorkspacePreset struct {
	Name       string            `mapstructure:"name"`
	Parameters map[string]string `mapstructure:"parameters"`
	Prebuilds  WorkspacePrebuild `mapstructure:"prebuilds"`
}

type WorkspacePrebuild struct {
	Instances int `mapstructure:"instances"`
}

func workspacePresetDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to predefine common configurations for coder workspaces. Users will have the option to select a defined preset, which will automatically apply the selected configuration. Any parameters defined in the preset will be applied to the workspace. Parameters that are not defined by the preset will still be configurable when creating a workspace.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			var preset WorkspacePreset
			err := mapstructure.Decode(struct {
				Name       interface{}
				Parameters interface{}
				Prebuilds  struct {
					Instances interface{}
				}
			}{
				Name:       rd.Get("name"),
				Parameters: rd.Get("parameters"),
				Prebuilds: struct {
					Instances interface{}
				}{
					Instances: rd.Get("prebuilds.0.instances"),
				},
			}, &preset)
			if err != nil {
				return diag.Errorf("decode workspace preset: %s", err)
			}

			rd.SetId(preset.Name)

			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Description: "The preset ID is automatically generated and may change between runs. It is recommended to use the `name` attribute to identify the preset.",
				Computed:    true,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "The name of the workspace preset.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "Workspace parameters that will be set by the workspace preset. For simple templates that only need prebuilds, you may define a preset with zero parameters. Because workspace parameters may change between Coder template versions, preset parameters are allowed to define values for parameters that do not exist in the current template version.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			"prebuilds": {
				Type:        schema.TypeSet,
				Description: "Prebuilt workspace configuration related to this workspace preset. Coder will build and maintain workspaces in reserve based on this configuration. When a user creates a new workspace using a preset, they will be assigned a prebuilt workspace, instead of waiting for a new workspace to build.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instances": {
							Type:         schema.TypeInt,
							Description:  "The number of workspaces to keep in reserve for this preset.",
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntAtLeast(0),
						},
					},
				},
			},
		},
	}
}
