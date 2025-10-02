package provider

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type AITask struct {
	ID         string             `mapstructure:"id"`
	SidebarApp []AITaskSidebarApp `mapstructure:"sidebar_app"`
	Prompt     string             `mapstructure:"prompt"`
	AppID      string             `mapstructure:"app_id"`
}

type AITaskSidebarApp struct {
	ID string `mapstructure:"id"`
}

// TaskPromptParameterName is the name of the parameter which is *required* to be defined when a coder_ai_task is used.
const TaskPromptParameterName = "AI Prompt"

func aiTaskResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to define Coder tasks.",
		CreateContext: func(c context.Context, resourceData *schema.ResourceData, i any) diag.Diagnostics {
			var diags diag.Diagnostics

			if idStr := os.Getenv("CODER_TASK_ID"); idStr != "" {
				resourceData.SetId(idStr)
			} else {
				resourceData.SetId(uuid.NewString())

				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "`CODER_TASK_ID` should be set. If you are seeing this message, the version of the Coder Terraform provider you are using is likely too new for your current Coder version.",
				})
			}

			if prompt := os.Getenv("CODER_TASK_PROMPT"); prompt != "" {
				resourceData.Set("prompt", prompt)
			} else {
				resourceData.Set("prompt", "default")
			}

			var (
				appID         = resourceData.Get("app_id").(string)
				sidebarAppSet = resourceData.Get("sidebar_app").(*schema.Set)
			)

			if appID == "" && sidebarAppSet.Len() > 0 {
				sidebarApps := sidebarAppSet.List()
				sidebarApp := sidebarApps[0].(map[string]any)

				if id, ok := sidebarApp["id"].(string); ok && id != "" {
					appID = id
					resourceData.Set("app_id", id)
				}
			}

			if appID == "" {
				return diag.Errorf("'app_id' must be set")
			}

			return diags
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Description: "A unique identifier for this resource.",
				Computed:    true,
			},
			"sidebar_app": {
				Type:          schema.TypeSet,
				Description:   "The coder_app to display in the sidebar. Usually a chat interface with the AI agent running in the workspace, like https://github.com/coder/agentapi.",
				Deprecated:    "This field has been deprecated in favor of the `app_id` field.",
				ForceNew:      true,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"app_id"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Description:  "A reference to an existing `coder_app` resource in your template.",
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IsUUID,
						},
					},
				},
			},
			"prompt": {
				Type:        schema.TypeString,
				Description: "The prompt text provided to the task by Coder.",
				Computed:    true,
			},
			"app_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the `coder_app` resource that provides the AI interface for this task.",
				ForceNew:      true,
				Optional:      true,
				Computed:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"sidebar_app"},
			},
		},
	}
}
