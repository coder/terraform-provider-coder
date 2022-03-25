package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type config struct {
	URL *url.URL
}

// New returns a new Terraform provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Description: "The URL to access Coder.",
				Optional:    true,
				// The "CODER_URL" environment variable is used by default
				// as the Access URL when generating scripts.
				DefaultFunc: schema.EnvDefaultFunc("CODER_URL", "https://mydeployment.coder.com"),
				ValidateFunc: func(i interface{}, s string) ([]string, []error) {
					_, err := url.Parse(s)
					if err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},
		},
		ConfigureContextFunc: func(c context.Context, resourceData *schema.ResourceData) (interface{}, diag.Diagnostics) {
			rawURL, ok := resourceData.Get("url").(string)
			if !ok {
				return nil, diag.Errorf("unexpected type %q for url", reflect.TypeOf(resourceData.Get("url")).String())
			}
			if rawURL == "" {
				return nil, diag.Errorf("CODER_URL must not be empty; got %q", rawURL)
			}
			parsed, err := url.Parse(resourceData.Get("url").(string))
			if err != nil {
				return nil, diag.FromErr(err)
			}
			return config{
				URL: parsed,
			}, nil
		},
		DataSourcesMap: map[string]*schema.Resource{
			"coder_workspace": {
				Description: "Use this data source to get information for the active workspace build.",
				ReadContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					rd.SetId(uuid.NewString())
					transition := os.Getenv("CODER_WORKSPACE_TRANSITION")
					if transition == "" {
						// Default to start!
						transition = "start"
					}
					rd.Set("transition", transition)
					rd.Set("owner", os.Getenv("CODER_WORKSPACE_OWNER"))
					rd.Set("name", os.Getenv("CODER_WORKSPACE_NAME"))
					return nil
				},
				Schema: map[string]*schema.Schema{
					"transition": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: `Either "start" or "stop". Use this to start/stop resources with "count".`,
					},
					"owner": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Username of the workspace owner.",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Name of the workspace.",
					},
				},
			},
			"coder_agent_script": {
				Description: "Use this data source to get the startup script to pull and start the Coder agent.",
				ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					config, valid := i.(config)
					if !valid {
						return diag.Errorf("config was unexpected type %q", reflect.TypeOf(i).String())
					}
					auth, valid := resourceData.Get("auth").(string)
					if !valid {
						return diag.Errorf("auth was unexpected type %q", reflect.TypeOf(resourceData.Get("auth")))
					}
					operatingSystem, valid := resourceData.Get("os").(string)
					if !valid {
						return diag.Errorf("os was unexpected type %q", reflect.TypeOf(resourceData.Get("os")))
					}
					arch, valid := resourceData.Get("arch").(string)
					if !valid {
						return diag.Errorf("arch was unexpected type %q", reflect.TypeOf(resourceData.Get("arch")))
					}
					accessURL, err := config.URL.Parse("/")
					if err != nil {
						return diag.Errorf("parse access url: %s", err)
					}
					script := os.Getenv(fmt.Sprintf("CODER_AGENT_SCRIPT_%s_%s", operatingSystem, arch))
					if script != "" {
						script = strings.ReplaceAll(script, "${ACCESS_URL}", accessURL.String())
						script = strings.ReplaceAll(script, "${AUTH_TYPE}", auth)
					}
					err = resourceData.Set("value", script)
					if err != nil {
						return diag.FromErr(err)
					}
					resourceData.SetId(strings.Join([]string{operatingSystem, arch}, "_"))
					return nil
				},
				Schema: map[string]*schema.Schema{
					"auth": {
						Type:         schema.TypeString,
						Default:      "token",
						Optional:     true,
						Description:  `The authentication type the agent will use. Must be one of: "token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity".`,
						ValidateFunc: validation.StringInSlice([]string{"token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity"}, false),
					},
					"os": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  `The operating system the agent will run on. Must be one of: "linux", "darwin", or "windows".`,
						ValidateFunc: validation.StringInSlice([]string{"linux", "darwin", "windows"}, false),
					},
					"arch": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  `The architecture the agent will run on. Must be one of: "amd64", "arm64".`,
						ValidateFunc: validation.StringInSlice([]string{"amd64"}, false),
					},
					"value": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"coder_agent": {
				Description: "Use this resource to associate an agent.",
				CreateContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					// This should be a real authentication token!
					rd.SetId(uuid.NewString())
					err := rd.Set("token", uuid.NewString())
					if err != nil {
						return diag.FromErr(err)
					}
					return nil
				},
				ReadContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				DeleteContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				Schema: map[string]*schema.Schema{
					"instance_id": {
						ForceNew:    true,
						Description: "An instance ID from a provisioned instance to enable zero-trust agent authentication.",
						Optional:    true,
						Type:        schema.TypeString,
					},
					"env": {
						ForceNew:    true,
						Description: "A mapping of environment variables to set inside the workspace.",
						Type:        schema.TypeMap,
						Optional:    true,
					},
					"startup_script": {
						ForceNew:    true,
						Description: "A script to run after the agent starts.",
						Type:        schema.TypeString,
						Optional:    true,
					},
					"token": {
						ForceNew:    true,
						Description: `Set the environment variable "CODER_TOKEN" with this token to authenticate an agent.`,
						Type:        schema.TypeString,
						Computed:    true,
					},
				},
			},
		},
	}
}
