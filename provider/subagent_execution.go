package provider

import (
	"context"
	"math"
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var subagentExecutionNameRegex = regexp.MustCompile(`(?i)^[a-z0-9](-?[a-z0-9])*$`)

func subagentExecutionResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Declares desired state for a runtime-agnostic child execution associated with a Coder agent. This provider resource does not launch or certify a sandbox. It requires a compatible Coder version that understands `coder_subagent_execution`.",
		CreateContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())

			if err := rd.Set("subagent_id", uuid.NewString()); err != nil {
				return diag.FromErr(err)
			}

			return nil
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: "The `id` property of the parent `coder_agent` resource.",
				ForceNew:    true,
				Required:    true,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "The name of the child execution. It may contain alphanumeric characters and single hyphens, and must begin and end with an alphanumeric character.",
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringMatch(subagentExecutionNameRegex, "must match (?i)^[a-z0-9](-?[a-z0-9])*$"),
			},
			"driver": {
				Type:         schema.TypeString,
				Description:  "The runtime driver configuration. This content is persisted in Terraform state and must not contain secrets.",
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"driver_protocol": {
				Type:         schema.TypeInt,
				Description:  "The protocol version used to interpret the driver configuration. Only version 1 is supported.",
				ForceNew:     true,
				Optional:     true,
				Default:      1,
				ValidateFunc: validation.IntInSlice([]int{1}),
			},
			"shared_host_path": {
				Type:         schema.TypeString,
				Description:  "The path on the parent host to share with the child execution.",
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"shared_child_path": {
				Type:         schema.TypeString,
				Description:  "The path in the child execution that receives the shared host path.",
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"startup_timeout": {
				Type:         schema.TypeInt,
				Description:  "Time in seconds allowed for the child execution to start.",
				ForceNew:     true,
				Optional:     true,
				Default:      120,
				ValidateFunc: validation.IntBetween(1, math.MaxInt32),
			},
			"restart_policy": {
				Type:         schema.TypeString,
				Description:  "Controls whether a failed child execution is restarted. Valid values are `never` and `on-failure`.",
				ForceNew:     true,
				Optional:     true,
				Default:      "on-failure",
				ValidateFunc: validation.StringInSlice([]string{"never", "on-failure"}, false),
			},
			"subagent_id": {
				Type:        schema.TypeString,
				Description: "The ID assigned to the declared child execution.",
				Computed:    true,
			},
		},
	}
}
