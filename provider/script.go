package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/robfig/cron/v3"
)

var ScriptCRONParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional | cron.Descriptor)

// ValidateCronExpression validates a cron expression and provides helpful warnings for common mistakes
func ValidateCronExpression(cronExpr string) (warnings []string, errors []error) {
	// Check if it looks like a 5-field Unix cron expression
	fields := strings.Fields(cronExpr)
	if len(fields) == 5 {
		// Try to parse as standard Unix cron (without seconds)
		unixParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional | cron.Descriptor)
		if _, err := unixParser.Parse(cronExpr); err == nil {
			// It's a valid 5-field expression, provide a helpful warning
			warnings = append(warnings, fmt.Sprintf(
				"The cron expression '%s' appears to be in Unix 5-field format. "+
					"Coder uses 6-field format (seconds minutes hours day month day-of-week). "+
					"Consider prefixing with '0 ' to run at the start of each minute: '0 %s'",
				cronExpr, cronExpr))
		}
	}

	// Validate with the actual 6-field parser
	_, err := ScriptCRONParser.Parse(cronExpr)
	if err != nil {
		errors = append(errors, fmt.Errorf("%s is not a valid cron expression: %w", cronExpr, err))
	}

	return warnings, errors
}

func scriptResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to run a script from an agent. When multiple scripts are assigned to the same agent, they are executed in parallel.",
		CreateContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())
			runOnStart, _ := rd.Get("run_on_start").(bool)
			startBlocksLogin, _ := rd.Get("start_blocks_login").(bool)
			runOnStop, _ := rd.Get("run_on_stop").(bool)
			cron, _ := rd.Get("cron").(string)

			if !runOnStart && !runOnStop && cron == "" {
				return diag.Errorf(`at least one of "run_on_start", "run_on_stop", or "cron" must be set`)
			}
			if !runOnStart && startBlocksLogin {
				return diag.Errorf(`"start_blocks_login" can only be set if "run_on_start" is "true"`)
			}
			return nil
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: "The `id` property of a `coder_agent` resource to associate with.",
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
					"icons [here](https://github.com/coder/coder/tree/main/site/static/icon). Use a " +
					"built-in icon with `\"${data.coder_workspace.me.access_url}/icon/<path>\"`.",
			},
			"script": {
				ForceNew:    true,
				Type:        schema.TypeString,
				Required:    true,
				Description: "The content of the script that will be run.",
			},
			"cron": {
				ForceNew:    true,
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The cron schedule to run the script on. This uses a 6-field cron expression format: `seconds minutes hours day-of-month month day-of-week`. Note that this differs from the standard Unix 5-field format by including seconds as the first field. Examples: `\"0 0 22 * * *\"` (daily at 10 PM), `\"0 */5 * * * *\"` (every 5 minutes), `\"30 0 9 * * 1-5\"` (weekdays at 9:30 AM).",
				ValidateFunc: func(i interface{}, _ string) ([]string, []error) {
					v, ok := i.(string)
					if !ok {
						return []string{}, []error{fmt.Errorf("got type %T instead of string", i)}
					}
					return ValidateCronExpression(v)
				},
			},
			"start_blocks_login": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "This option determines whether users can log in immediately or must wait for the workspace to finish running this script upon startup. If not enabled, users may encounter an incomplete workspace when logging in. This option only sets the default, the user can still manually override the behavior.",
			},
			"run_on_start": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "This option defines whether or not the script should run when the agent starts. The script should exit when it is done to signal that the agent is ready.",
			},
			"run_on_stop": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "This option defines whether or not the script should run when the agent stops. The script should exit when it is done to signal that the workspace can be stopped.",
			},
			"timeout": {
				Type:         schema.TypeInt,
				Default:      0,
				ForceNew:     true,
				Optional:     true,
				Description:  "Time in seconds that the script is allowed to run. If the script does not complete within this time, the script is terminated and the agent lifecycle status is marked as timed out. A value of zero (default) means no timeout.",
				ValidateFunc: validation.IntAtLeast(1),
			},
		},
	}
}
