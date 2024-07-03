package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/provider/helpers"
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

			owner := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_OWNER", "default")
			_ = rd.Set("owner", owner)

			ownerEmail := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_OWNER_EMAIL", "default@example.com")
			_ = rd.Set("owner_email", ownerEmail)

			ownerGroupsText := helpers.OptionalEnv("CODER_WORKSPACE_OWNER_GROUPS")
			var ownerGroups []string
			if ownerGroupsText != "" {
				err := json.Unmarshal([]byte(ownerGroupsText), &ownerGroups)
				if err != nil {
					return diag.Errorf("couldn't parse owner groups %q", ownerGroupsText)
				}
			}
			_ = rd.Set("owner_groups", ownerGroups)

			ownerName := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_OWNER_NAME", "default")
			_ = rd.Set("owner_name", ownerName)

			ownerID := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_OWNER_ID", uuid.Nil.String())
			_ = rd.Set("owner_id", ownerID)

			ownerOIDCAccessToken := helpers.OptionalEnv("CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN")
			_ = rd.Set("owner_oidc_access_token", ownerOIDCAccessToken)

			name := helpers.OptionalEnvOrDefault("CODER_WORKSPACE_NAME", "default")
			rd.Set("name", name)

			sessionToken := helpers.OptionalEnv("CODER_WORKSPACE_OWNER_SESSION_TOKEN")
			_ = rd.Set("owner_session_token", sessionToken)

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
