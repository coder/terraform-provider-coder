package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type AITask struct {
	ID         string             `mapstructure:"id"`
	SidebarApp []AITaskSidebarApp `mapstructure:"sidebar_app"`
}

type AITaskSidebarApp struct {
	ID string `mapstructure:"id"`
}

// TaskPromptParameterName is the name of the parameter which is *required* to be defined when a coder_ai_task is used.
const TaskPromptParameterName = "AI Prompt"

func aiTask() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

                Description: "Use this resource to define Coder tasks. @since:v2.24.0",
		CreateContext: func(c context.Context, resourceData *schema.ResourceData, i any) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())
			return nil
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
				Type:        schema.TypeSet,
				Description: "The coder_app to display in the sidebar. Usually a chat interface with the AI agent running in the workspace, like https://github.com/coder/agentapi.",
				ForceNew:    true,
				Required:    true,
				MaxItems:    1,
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
		},
	}
}
