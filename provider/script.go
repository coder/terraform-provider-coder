package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func scriptResource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this resource to run a script from an agent.",
		CreateContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())
			return nil
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: `The "id" property of a "coder_agent" resource to associate with.`,
				ForceNew:    true,
				Required:    true,
			},
			"display_name": {
				Type:        schema.TypeString,
				Description: "The display name of the script to display logs in the dashboard.",
				ForceNew:    true,
				Required:    true,
			},
			"log_path": {
				Type:        schema.TypeString,
				Description: "The path of a file to write the logs to. If relative, it will be appended to tmp.",
				ForceNew:    true,
				Optional:    true,
			},
			"icon": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Description: "A URL to an icon that will display in the dashboard. View built-in " +
					"icons here: https://github.com/coder/coder/tree/main/site/static/icon. Use a " +
					"built-in icon with `data.coder_workspace.me.access_url + \"/icon/<path>\"`.",
			},
			"script": {
				ForceNew:    true,
				Type:        schema.TypeString,
				Required:    true,
				Description: "The script to run.",
			},
			"cron": {
				ForceNew:    true,
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The cron schedule to run the script on. This is a cron expression.",
			},
			"start_blocks_login": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "This option defines whether or not the user can (by default) login to the workspace before this script completes running on start. When enabled, users may see an incomplete workspace when logging in.",
			},
			"run_on_start": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "This option defines whether or not the script should run when the agent starts.",
			},
			"run_on_stop": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "This option defines whether or not the script should run when the agent stops.",
			},
			"timeout": {
				Type:         schema.TypeInt,
				Default:      0,
				ForceNew:     true,
				Optional:     true,
				Description:  "Time in seconds until the agent lifecycle status is marked as timed out, this happens when the script has not completed (exited) in the given time.",
				ValidateFunc: validation.IntAtLeast(1),
			},
		},
	}
}
