package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

type Monitoring struct {
	Threshold       int
	MemoryThreshold int
	DiskThreshold   int
	Disks           []string
	Enabled         bool
	MemoryEnabled   bool
	DiskEnabled     bool
	AgentID         string
	Validation      []Validation
}

func monitoringDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to configure editable options for workspaces.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())

			fixedValidation, err := fixValidationResourceData(rd.GetRawConfig(), rd.Get("validation"))
			if err != nil {
				return diag.FromErr(err)
			}

			err = rd.Set("validation", fixedValidation)
			if err != nil {
				return diag.FromErr(err)
			}

			var monitoring Monitoring
			err = mapstructure.Decode(struct {
				Threshold       interface{}
				MemoryThreshold interface{}
				DiskThreshold   interface{}
				Disks           interface{}
				Enabled         interface{}
				MemoryEnabled   interface{}
				DiskEnabled     interface{}
				AgentID         interface{}
				Validation      interface{}
			}{
				Threshold:       rd.Get("threshold"),
				MemoryThreshold: rd.Get("memory_threshold"),
				DiskThreshold:   rd.Get("disk_threshold"),
				Disks:           rd.Get("disks"),
				Enabled:         rd.Get("enabled"),
				MemoryEnabled:   rd.Get("memory_enabled"),
				DiskEnabled:     rd.Get("disk_enabled"),
				AgentID:         rd.Get("agent_id"),
				Validation:      fixedValidation,
			}, &monitoring)
			if err != nil {
				return diag.FromErr(err)
			}

			return nil
		},
		Schema: map[string]*schema.Schema{
			"threshold": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The threshold for the monitoring module.",
			},
			"memory_threshold": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The memory threshold for the monitoring module.",
			},
			"disk_threshold": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The disk threshold for the monitoring module.",
			},
			"disks": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    10,
				Description: "The disks to monitor.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the monitoring module is enabled.",
			},
			"memory_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the memory monitoring module is enabled.",
			},
			"disk_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the disk monitoring module is enabled.",
			},
			"agent_id": {
				Type:        schema.TypeString,
				Description: "The ID of the agent to use for the monitoring module.",
				ForceNew:    true,
				Optional:    false,
			},
			"validation": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Validate the input of a parameter.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The minimum of a number parameter.",
						},
						"min_disabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Helper field to check if min is present",
						},
						"max": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The maximum of a number parameter.",
						},
						"max_disabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Helper field to check if max is present",
						},
						"monotonic": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Number monotonicity, either increasing or decreasing.",
						},
						"regex": {
							Type:          schema.TypeString,
							ConflictsWith: []string{"validation.0.min", "validation.0.max", "validation.0.monotonic"},
							Description:   "A regex for the input parameter to match against.",
							Optional:      true,
						},
						"error": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An error message to display if the value breaks the validation rules. The following placeholders are supported: {max}, {min}, and {value}.",
						},
					},
				},
			},
		},
	}
}
