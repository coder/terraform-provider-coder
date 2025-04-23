package provider

import (
	"context"
	"reflect"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

func workspaceDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to get information for the active workspace build.",
		ReadContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			transition := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_TRANSITION", "start") // Default to start!
			_ = rd.Set("transition", transition)

			count := 0
			if transition == "start" {
				count = 1
			}
			_ = rd.Set("start_count", count)

			isPrebuild := isPrebuiltWorkspace(c)
			prebuildCount := 0
			if isPrebuild {
				prebuildCount = 1
				_ = rd.Set("is_prebuild", true)
			}
			_ = rd.Set("prebuild_count", prebuildCount)

			name := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_NAME", "default")
			rd.Set("name", name)

			id := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_ID", uuid.NewString())
			rd.SetId(id)

			templateID, err := helpers.RequireEnv("CODER_WORKSPACE_TEMPLATE_ID")
			if err != nil {
				return diag.Errorf("template ID is missing: %s", err.Error())
			}
			_ = rd.Set("template_id", templateID)

			templateName, err := helpers.RequireEnv("CODER_WORKSPACE_TEMPLATE_NAME")
			if err != nil {
				return diag.Errorf("template name is missing: %s", err.Error())
			}
			_ = rd.Set("template_name", templateName)

			templateVersion, err := helpers.RequireEnv("CODER_WORKSPACE_TEMPLATE_VERSION")
			if err != nil {
				return diag.Errorf("template version is missing: %s", err.Error())
			}
			_ = rd.Set("template_version", templateVersion)

			config, valid := i.(config)
			if !valid {
				return diag.Errorf("config was unexpected type %q", reflect.TypeOf(i).String())
			}
			rd.Set("access_url", config.URL.String())

			rawPort := config.URL.Port()
			if rawPort == "" {
				rawPort = "80"
				if config.URL.Scheme == "https" {
					rawPort = "443"
				}
			}
			port, err := strconv.Atoi(rawPort)
			if err != nil {
				return diag.Errorf("couldn't parse port %q", port)
			}
			rd.Set("access_port", port)

			return nil
		},
		Schema: map[string]*schema.Schema{
			"access_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access URL of the Coder deployment provisioning this workspace.",
			},
			"access_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The access port of the Coder deployment provisioning this workspace.",
			},
			"prebuild_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "A computed count, equal to 1 if the workspace is a currently unassigned prebuild. Use this to conditionally act on the status of a prebuild. Actions that do not require user identity can be taken when this value is set to 1. Actions that should only be taken once the workspace has been assigned to a user may be taken when this value is set to 0.",
			},
			"start_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "A computed count based on `transition` state. If `start`, count will equal 1.",
			},
			"transition": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Either `start` or `stop`. Use this to start/stop resources with `count`.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "UUID of the workspace.",
			},
			"is_prebuild": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Similar to `prebuild_count`, but a boolean value instead of a count. This is set to true if the workspace is a currently unassigned prebuild. Once the workspace is assigned, this value will be false.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the workspace.",
			},
			"template_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the workspace's template.",
			},
			"template_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the workspace's template.",
			},
			"template_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Version of the workspace's template.",
			},
		},
	}
}

func isPrebuiltWorkspace(ctx context.Context) bool {
	return helpers.OptionalEnv(IsPrebuildEnvironmentVariable()) == "true"
}

// IsPrebuildEnvironmentVariable returns the name of the environment
// variable that indicates whether the workspace is an unclaimed prebuilt workspace.
// The value of this environment variable should be set to "true" if the workspace
// is prebuilt and it has not yet been claimed by a user.
// Any other values, including "false" and "" will be interpreted to mean that the
// workspace is not prebuilt, or was prebuilt but has since been claimed by a user.
func IsPrebuildEnvironmentVariable() string {
	return "CODER_WORKSPACE_IS_PREBUILD"
}
