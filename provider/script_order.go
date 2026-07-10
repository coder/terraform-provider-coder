package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func scriptOrderResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to declare ordering constraints between `coder_script` resources on an agent. Multiple `coder_script_order` resources are additive; each `rule` block adds edges to the startup dependency graph. Reference scripts by their id, e.g. `coder_script.install.id`.",
		CreateContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())
			// Validate that every rule declares at least one script to run.
			rules, _ := rd.Get("rule").([]interface{})
			for i, raw := range rules {
				m, _ := raw.(map[string]interface{})
				run, _ := m["run"].([]interface{})
				if len(run) == 0 {
					return diag.Errorf("rule[%d]: \"run\" must contain at least one coder_script id", i)
				}
			}
			return nil
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"rule": {
				Type:        schema.TypeList,
				Description: "Each `rule` block declares that the scripts in `run` must wait for the scripts in `after` to reach `state`.",
				ForceNew:    true,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"run": {
							Type:        schema.TypeList,
							Description: "The ids of the `coder_script` resources this rule applies to (e.g. `coder_script.install.id`).",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"after": {
							Type:        schema.TypeList,
							Description: "The ids of the `coder_script` resources that must reach `state` before the `run` scripts start.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"state": {
							Type:         schema.TypeString,
							Description:  "The lifecycle state the `after` scripts must reach before the `run` scripts may start. One of `completes` or `starts`.",
							Optional:     true,
							Default:      "completes",
							ValidateFunc: validation.StringInSlice([]string{"completes", "starts"}, false),
						},
					},
				},
			},
		},
	}
}
