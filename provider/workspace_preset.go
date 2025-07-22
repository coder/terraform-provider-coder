package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/mapstructure"
	rbcron "github.com/robfig/cron/v3"
)

var PrebuildsCRONParser = rbcron.NewParser(rbcron.Minute | rbcron.Hour | rbcron.Dom | rbcron.Month | rbcron.Dow)

type WorkspacePreset struct {
	Name        string            `mapstructure:"name"`
	Description string            `mapstructure:"description"`
	Icon        string            `mapstructure:"icon"`
	Default     bool              `mapstructure:"default"`
	Parameters  map[string]string `mapstructure:"parameters"`
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
	Scheduling       []Scheduling       `mapstructure:"scheduling"`
}

type ExpirationPolicy struct {
	TTL int `mapstructure:"ttl"`
}

type Scheduling struct {
	Timezone string     `mapstructure:"timezone"`
	Schedule []Schedule `mapstructure:"schedule"`
}

type Schedule struct {
	Cron      string `mapstructure:"cron"`
	Instances int    `mapstructure:"instances"`
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

			// Validate schedule overlaps if scheduling is configured
			err = validateSchedules(rd)
			if err != nil {
				return diag.Errorf("schedules overlap with each other: %s", err)
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
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Describe what this preset does.",
				ValidateFunc: validation.StringLenBetween(0, 128),
			},
			"icon": {
				Type: schema.TypeString,
				Description: "A URL to an icon that will display in the dashboard. View built-in " +
					"icons [here](https://github.com/coder/coder/tree/main/site/static/icon). Use a " +
					"built-in icon with `\"${data.coder_workspace.me.access_url}/icon/<path>\"`.",
				ForceNew: true,
				Optional: true,
				ValidateFunc: validation.All(
					helpers.ValidateURL,
					validation.StringLenBetween(0, 256),
				),
			},
			"default": {
				Type:        schema.TypeBool,
				Description: "Whether this preset should be selected by default when creating a workspace. Only one preset per template can be marked as default.",
				Optional:    true,
				Default:     false,
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
										// Ensure TTL is either 0 (to disable expiration) or between 3600 seconds (1 hour) and 31536000 seconds (1 year)
										ValidateFunc: func(val interface{}, key string) ([]string, []error) {
											v := val.(int)
											if v == 0 {
												return nil, nil
											}
											if v < 3600 || v > 31536000 {
												return nil, []error{fmt.Errorf("%q must be 0 or between 3600 and 31536000, got %d", key, v)}
											}
											return nil, nil
										},
									},
								},
							},
						},
						"scheduling": {
							Type:        schema.TypeList,
							Description: "Configuration block that defines scheduling behavior for prebuilds. Use this to automatically adjust the number of prebuild instances based on a schedule.",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"timezone": {
										Type: schema.TypeString,
										Description: `The timezone to use for the prebuild schedules (e.g., "UTC", "America/New_York"). 
Timezone must be a valid timezone in the IANA timezone database. 
See https://en.wikipedia.org/wiki/List_of_tz_database_time_zones for a complete list of valid timezone identifiers and https://www.iana.org/time-zones for the official IANA timezone database.`,
										Required: true,
										ValidateFunc: func(val interface{}, key string) ([]string, []error) {
											timezone := val.(string)

											_, err := time.LoadLocation(timezone)
											if err != nil {
												return nil, []error{fmt.Errorf("failed to load timezone %q: %w", timezone, err)}
											}

											return nil, nil
										},
									},
									"schedule": {
										Type:        schema.TypeList,
										Description: "One or more schedule blocks that define when to scale the number of prebuild instances.",
										Required:    true,
										MinItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"cron": {
													Type:        schema.TypeString,
													Description: "A cron expression that defines when this schedule should be active. The cron expression must be in the format \"* HOUR DOM MONTH DAY-OF-WEEK\" where HOUR is 0-23, DOM (day-of-month) is 1-31, MONTH is 1-12, and DAY-OF-WEEK is 0-6 (Sunday-Saturday). The minute field must be \"*\" to ensure the schedule covers entire hours rather than specific minute intervals.",
													Required:    true,
													ValidateFunc: func(val interface{}, key string) ([]string, []error) {
														cronSpec := val.(string)

														err := validatePrebuildsCronSpec(cronSpec)
														if err != nil {
															return nil, []error{fmt.Errorf("cron spec failed validation: %w", err)}
														}

														_, err = PrebuildsCRONParser.Parse(cronSpec)
														if err != nil {
															return nil, []error{fmt.Errorf("failed to parse cron spec: %w", err)}
														}

														return nil, nil
													},
												},
												"instances": {
													Type:        schema.TypeInt,
													Description: "The number of prebuild instances to maintain during this schedule period.",
													Required:    true,
												},
											},
										},
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

// validatePrebuildsCronSpec ensures that the minute field is set to *.
// This is required because prebuild schedules represent continuous time ranges,
// and we want the schedule to cover entire hours rather than specific minute intervals.
func validatePrebuildsCronSpec(spec string) error {
	parts := strings.Fields(spec)
	if len(parts) != 5 {
		return fmt.Errorf("cron specification should consist of 5 fields")
	}
	if parts[0] != "*" {
		return fmt.Errorf("minute field should be *")
	}

	return nil
}

// validateSchedules checks if any of the configured prebuild schedules overlap with each other.
// It returns an error if overlaps are found, nil otherwise.
func validateSchedules(rd *schema.ResourceData) error {
	// TypeSet from schema definition
	prebuilds := rd.Get("prebuilds").(*schema.Set)
	if prebuilds.Len() == 0 {
		return nil
	}

	// Each element of TypeSet with Elem: &schema.Resource{} should be map[string]interface{}
	prebuild, ok := prebuilds.List()[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid prebuild configuration: expected map[string]interface{}")
	}

	// TypeList from schema definition
	schedulingBlocks, ok := prebuild["scheduling"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid scheduling configuration: expected []interface{}")
	}
	if len(schedulingBlocks) == 0 {
		return nil
	}

	// Each element of TypeList with Elem: &schema.Resource{} should be map[string]interface{}
	schedulingBlock, ok := schedulingBlocks[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid scheduling configuration: expected map[string]interface{}")
	}

	// TypeList from schema definition
	scheduleBlocks, ok := schedulingBlock["schedule"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid schedule configuration: expected []interface{}")
	}
	if len(scheduleBlocks) == 0 {
		return nil
	}

	cronSpecs := make([]string, len(scheduleBlocks))
	for i, scheduleBlock := range scheduleBlocks {
		// Each element of TypeList with Elem: &schema.Resource{} should be map[string]interface{}
		schedule, ok := scheduleBlock.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid schedule configuration: expected map[string]interface{}")
		}

		// TypeString from schema definition
		cronSpec := schedule["cron"].(string)

		cronSpecs[i] = cronSpec
	}

	err := helpers.ValidateSchedules(cronSpecs)
	if err != nil {
		return err
	}

	return nil
}
