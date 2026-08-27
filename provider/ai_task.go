package provider

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Deprecated: Coder Tasks is deprecated as of Coder v2.36 and will be removed in
// a future release. Use Coder Agents instead:
// https://coder.com/docs/ai-coder/agents/tasks-to-chats-migration
type AITask struct {
	ID         string             `mapstructure:"id"`
	SidebarApp []AITaskSidebarApp `mapstructure:"sidebar_app"`
	Prompt     string             `mapstructure:"prompt"`
	AppID      string             `mapstructure:"app_id"`
}

// Deprecated: Coder Tasks is deprecated as of Coder v2.36 and will be removed in
// a future release. Use Coder Agents instead:
// https://coder.com/docs/ai-coder/agents/tasks-to-chats-migration
type AITaskSidebarApp struct {
	ID string `mapstructure:"id"`
}

// TaskPromptParameterName is the name of the parameter which is *required* to be defined when a coder_ai_task is used.
//
// Deprecated: Coder Tasks is deprecated as of Coder v2.36. Task prompts are read
// from the task itself, not from a parameter.
const TaskPromptParameterName = "AI Prompt"

// aiTaskDeprecationMessage is surfaced by Terraform whenever a deprecated AI
// task resource or data source is used.
const aiTaskDeprecationMessage = "Coder Tasks is deprecated as of Coder v2.36 and will be removed in a future release. Use Coder Agents instead: https://coder.com/docs/ai-coder/agents/tasks-to-chats-migration"

// aiTaskAttributeDeprecationMessage is surfaced by Terraform whenever a
// deprecated AI task attribute is used. It is kept short because it is rendered
// inline in the generated attribute documentation.
const aiTaskAttributeDeprecationMessage = "Coder Tasks is deprecated as of Coder v2.36 and will be removed in a future release."

// aiTaskDeprecationNotice is rendered in the generated documentation.
const aiTaskDeprecationNotice = "\n\n~> **Deprecated**: Coder Tasks is deprecated as of Coder v2.36 and will be removed in a future release. Templates no longer require AI task resources; use [Coder Agents](https://coder.com/docs/ai-coder/agents) and follow the [migration guide](https://coder.com/docs/ai-coder/agents/tasks-to-chats-migration)."

func aiTaskResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		DeprecationMessage: aiTaskDeprecationMessage,
		Description:        "Use this resource to define Coder tasks." + aiTaskDeprecationNotice,
		CreateContext: func(c context.Context, resourceData *schema.ResourceData, i any) diag.Diagnostics {
			var diags diag.Diagnostics

			if id, err := uuid.Parse(os.Getenv("CODER_TASK_ID")); err == nil && id != uuid.Nil {
				resourceData.SetId(id.String())
				resourceData.Set("enabled", true)
			} else {
				resourceData.SetId(uuid.NewString())
				resourceData.Set("enabled", false)
			}

			if prompt := os.Getenv("CODER_TASK_PROMPT"); prompt != "" {
				resourceData.Set("prompt", prompt)
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
				Deprecated:    "This field has been deprecated in favor of the `app_id` field. " + aiTaskAttributeDeprecationMessage,
				ForceNew:      true,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"app_id"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Description:  "A reference to an existing `coder_app` resource in your template.",
							Deprecated:   aiTaskAttributeDeprecationMessage,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IsUUID,
						},
					},
				},
			},
			"prompt": {
				Type:        schema.TypeString,
				Description: "The prompt text provided to the task by Coder.\n\n  -> The `prompt` field is only populated in Coder v2.28 and later.",
				Deprecated:  aiTaskAttributeDeprecationMessage,
				Computed:    true,
			},
			"app_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the `coder_app` resource that provides the AI interface for this task.",
				Deprecated:    aiTaskAttributeDeprecationMessage,
				ForceNew:      true,
				Optional:      true,
				Computed:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"sidebar_app"},
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "True when executing in a Coder Task context, false when in a Coder Workspace context.\n\n  -> The `enabled` field is only populated in Coder v2.28 and later.",
				Deprecated:  aiTaskAttributeDeprecationMessage,
				Computed:    true,
			},
		},
	}
}

func taskDatasource() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: aiTaskDeprecationMessage,
		Description:        "Use this data source to read information about Coder Tasks." + aiTaskDeprecationNotice,
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			diags := diag.Diagnostics{}

			idStr := os.Getenv("CODER_TASK_ID")
			if idStr == "" || idStr == uuid.Nil.String() {
				rd.SetId(uuid.NewString())
				_ = rd.Set("enabled", false)
			} else if _, err := uuid.Parse(idStr); err == nil {
				rd.SetId(idStr)
				_ = rd.Set("enabled", true)
			} else { // invalid UUID
				diags = append(diags, errorAsDiagnostics(err)...)
			}

			_ = rd.Set("prompt", os.Getenv("CODER_TASK_PROMPT"))
			return diags
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the task, if executing in a Coder Task context. Empty in a Coder Workspace context.",
			},
			"prompt": {
				Type:        schema.TypeString,
				Computed:    true,
				Deprecated:  aiTaskAttributeDeprecationMessage,
				Description: "The prompt text provided to the task by Coder, if executing in a Coder Task context. Empty in a Coder Workspace context.\n\n  -> The `prompt` field is only populated in Coder v2.28 and later.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Deprecated:  aiTaskAttributeDeprecationMessage,
				Description: "True when executing in a Coder Task context, false when in a Coder Workspace context.\n\n  -> The `enabled` field is only populated in Coder v2.28 and later.",
			},
		},
	}
}
