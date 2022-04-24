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
					transition := os.Getenv("CODER_WORKSPACE_TRANSITION")
					if transition == "" {
						// Default to start!
						transition = "start"
					}
					_ = rd.Set("transition", transition)
					count := 0
					if transition == "start" {
						count = 1
					}
					_ = rd.Set("start_count", count)
					owner := os.Getenv("CODER_WORKSPACE_OWNER")
					if owner == "" {
						owner = "default"
					}
					_ = rd.Set("owner", owner)
					ownerID := os.Getenv("CODER_WORKSPACE_OWNER_ID")
					if ownerID == "" {
						ownerID = uuid.Nil.String()
					}
					_ = rd.Set("owner_id", ownerID)
					name := os.Getenv("CODER_WORKSPACE_NAME")
					if name == "" {
						name = "default"
					}
					rd.Set("name", name)
					id := os.Getenv("CODER_WORKSPACE_ID")
					if id == "" {
						id = uuid.NewString()
					}
					rd.SetId(id)
					return nil
				},
				Schema: map[string]*schema.Schema{
					"start_count": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: `A computed count based on "transition" state. If "start", count will equal 1.`,
					},
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
					"owner_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "UUID of the workspace owner.",
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
				},
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"coder_agent": {
				Description: "Use this resource to associate an agent.",
				CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					// This should be a real authentication token!
					resourceData.SetId(uuid.NewString())
					err := resourceData.Set("token", uuid.NewString())
					if err != nil {
						return diag.FromErr(err)
					}
					return updateInitScript(resourceData, i)
				},
				ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					return updateInitScript(resourceData, i)
				},
				DeleteContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				Schema: map[string]*schema.Schema{
					"init_script": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Run this script on startup of an instance to initialize the agent.",
					},
					"arch": {
						Type:         schema.TypeString,
						ForceNew:     true,
						Required:     true,
						Description:  `The architecture the agent will run on. Must be one of: "amd64", "arm64".`,
						ValidateFunc: validation.StringInSlice([]string{"amd64", "arm64"}, false),
					},
					"auth": {
						Type:         schema.TypeString,
						Default:      "token",
						ForceNew:     true,
						Optional:     true,
						Description:  `The authentication type the agent will use. Must be one of: "token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity".`,
						ValidateFunc: validation.StringInSlice([]string{"token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity"}, false),
					},
					"env": {
						ForceNew:    true,
						Description: "A mapping of environment variables to set inside the workspace.",
						Type:        schema.TypeMap,
						Optional:    true,
					},
					"os": {
						Type:         schema.TypeString,
						ForceNew:     true,
						Required:     true,
						Description:  `The operating system the agent will run on. Must be one of: "linux", "darwin", or "windows".`,
						ValidateFunc: validation.StringInSlice([]string{"linux", "darwin", "windows"}, false),
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
			"coder_agent_instance": {
				Description: "Use this resource to associate an instance ID with an agent for zero-trust " +
					"authentication. This association is done automatically for \"google_compute_instance\", " +
					"\"aws_instance\", \"azurerm_linux_virtual_machine\", and " +
					"\"azurerm_windows_virtual_machine\" resources.",
				CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					resourceData.SetId(uuid.NewString())
					return nil
				},
				ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				DeleteContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				Schema: map[string]*schema.Schema{
					"agent_id": {
						Type:        schema.TypeString,
						Description: `The "id" property of a "coder_agent" resource to associate with.`,
						ForceNew:    true,
						Required:    true,
					},
					"instance_id": {
						ForceNew:    true,
						Required:    true,
						Description: `The instance identifier of a provisioned resource.`,
						Type:        schema.TypeString,
					},
				},
			},
		},
	}
}

// updateInitScript fetches parameters from a "coder_agent" to produce the
// agent script from environment variables.
func updateInitScript(resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
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
	err = resourceData.Set("init_script", script)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
