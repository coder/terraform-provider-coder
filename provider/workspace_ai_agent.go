package provider

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func workspaceAIAgentDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to fetch the workspace's AI agent identity. " +
			"The session token is a scoped API key for the workspace's AI agent " +
			"identity, sponsored by the workspace owner. Template authors decide " +
			"where to inject it; typically it is passed to an AI tool or sandbox " +
			"so the tool can authenticate to the Coder AI gateway, including its " +
			"MCP endpoint. The token carries only the AI agent identity's " +
			"restricted scopes, never the owner's full permissions.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			if idStr := os.Getenv("CODER_WORKSPACE_AI_AGENT_ID"); idStr != "" {
				rd.SetId(idStr)
			} else {
				rd.SetId(uuid.NewString())
			}
			_ = rd.Set("session_token", os.Getenv("CODER_WORKSPACE_AI_AGENT_SESSION_TOKEN"))
			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the workspace's AI agent identity.",
			},
			"session_token": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Scoped session token for the workspace's AI agent identity. " +
					"It is regenerated every time the workspace is started. Empty when " +
					"the deployment does not provision an AI agent identity for this " +
					"workspace.",
				Sensitive: true,
			},
		},
	}
}
