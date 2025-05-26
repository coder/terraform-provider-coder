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
	// There should always be only one prebuild block, but Terraform's type system
	// still parses them as a slice, so we need to handle it as such. We could use
	// an anonymous type and rd.Get to avoid a slice here, but that would not be possible
	// for utilities that parse our terraform output using this type. To remain compatible
	// with those cases, we use a slice here.
	Prebuilds []WorkspacePrebuild `mapstructure:"prebuilds"`
}

type WorkspacePrebuild struct {
	Instances int `mapstructure:"instances"`
	// There should always be only one expiration_policy block, but Terraform's type system
	// still parses them as a slice, so we need to handle it as such. We could use
	// an anonymous type and rd.Get to avoid a slice here, but that would not be possible
	// for utilities that parse our terraform output using this type. To remain compatible
	// with those cases, we use a slice here.
	ExpirationPolicy []ExpirationPolicy `mapstructure:"expiration_policy"`
}

type ExpirationPolicy struct {
	TTL int `mapstructure:"ttl"`
}

func workspacePresetDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to predefine common configurations for coder workspaces. Users will have the option to select a defined preset, which will automatically apply the selected configuration. Any parameters defined in the preset will be applied to the workspace. Parameters that are defined by the template but not defined by the preset will still be configurable when creating a workspace.",

		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			var preset WorkspacePreset
			err := mapstructure.Decode(struct {
				Name interface{}
			}{
				Name: rd.Get("name"),
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
				Description: "Configuration for prebuilt workspaces associated with this preset. Coder will maintain a pool of standby workspaces based on this configuration. When a user creates a workspace using this preset, they are assigned a prebuilt workspace instead of waiting for a new one to build. See prebuilt workspace documentation [here](https://coder.com/docs/admin/templates/extending-templates/prebuilt-workspaces.md)",
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
						"expiration_policy": {
							Type:        schema.TypeSet,
							Description: "Configuration block that defines TTL (time-to-live) behavior for prebuilds. Use this to automatically invalidate and delete prebuilds after a certain period, ensuring they stay up-to-date.",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ttl": {
										Type:        schema.TypeInt,
										Description: "Time in seconds after which an unclaimed prebuild is considered expired and eligible for cleanup.",
										Required:    true,
										ForceNew:    true,
										// Ensure TTL is between 3600 seconds (1 hour) and 31536000 seconds (1 year)
										ValidateFunc: validation.IntBetween(3600, 31536000),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
