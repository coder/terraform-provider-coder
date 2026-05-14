package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DLPPolicy struct {
	ID                   string   `mapstructure:"id"`
	SSHAccess            bool     `mapstructure:"ssh_access"`
	WebTerminalAccess    bool     `mapstructure:"web_terminal_access"`
	PortForwardingAccess bool     `mapstructure:"port_forwarding_access"`
	DesktopAccess        bool     `mapstructure:"desktop_access"`
	ClipboardAccess      bool     `mapstructure:"clipboard_access"`
	AllowedApplications  []string `mapstructure:"allowed_applications"`
}

func dlpPolicyResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to declare a data loss prevention policy. " +
			"Declare at most one `coder_dlp_policy` resource per template; it applies " +
			"uniformly to every agent in the resulting workspace.",
		CreateContext: func(_ context.Context, rd *schema.ResourceData, _ any) diag.Diagnostics {
			rd.SetId(uuid.NewString())
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
			"ssh_access": {
				Type:        schema.TypeBool,
				Description: "Whether workspace users may connect to the workspace over SSH.",
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"web_terminal_access": {
				Type:        schema.TypeBool,
				Description: "Whether workspace users may open the in-browser web terminal.",
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"port_forwarding_access": {
				Type:        schema.TypeBool,
				Description: "Whether workspace users may forward arbitrary TCP ports from the workspace.",
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"desktop_access": {
				Type:        schema.TypeBool,
				Description: "Whether workspace users may open the noVNC desktop viewer.",
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"clipboard_access": {
				Type:        schema.TypeBool,
				Description: "Whether workspace users may use clipboard copy/paste in the noVNC desktop viewer. When false, ClientCutText and ServerCutText RFB messages are dropped in flight.",
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"allowed_applications": {
				Type: schema.TypeList,
				Description: "Slugs of coder_app resources workspace users are allowed to access. " +
					"Apps whose slugs are not in this list are blocked.",
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}
