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

		Description: "Use this data source to predefine common configurations for workspaces.",
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
				Description: "ID of the workspace preset.",
				Computed:    true,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "Name of the workspace preset.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "Parameters of the workspace preset.",
				Required:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			"prebuilds": {
				Type:        schema.TypeSet,
				Description: "Prebuilds of the workspace preset.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instances": {
							Type:         schema.TypeInt,
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
