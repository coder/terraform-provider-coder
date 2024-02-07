package provider

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/xerrors"
)

func agentResource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this resource to associate an agent.",
		CreateContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			// This should be a real authentication token!
			resourceData.SetId(uuid.NewString())
			err := resourceData.Set("token", uuid.NewString())
			if err != nil {
				return diag.FromErr(err)
			}

			if _, ok := resourceData.GetOk("display_apps"); !ok {
				err = resourceData.Set("display_apps", []interface{}{
					map[string]bool{
						"vscode":                 true,
						"vscode_insiders":        false,
						"web_terminal":           true,
						"ssh_helper":             true,
						"port_forwarding_helper": true,
					},
				})
				if err != nil {
					return diag.FromErr(err)
				}
			}

			rawPlan := resourceData.GetRawPlan()
			items := rawPlan.GetAttr("metadata").AsValueSlice()
			itemKeys := map[string]struct{}{}
			for _, item := range items {
				key := valueAsString(item.GetAttr("key"))
				_, exists := itemKeys[key]
				if exists {
					return diag.FromErr(xerrors.Errorf("duplicate agent metadata key %q", key))
				}
				itemKeys[key] = struct{}{}
			}

			return updateInitScript(resourceData, i)
		},
		ReadWithoutTimeout: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			err := resourceData.Set("token", uuid.NewString())
			if err != nil {
				return diag.FromErr(err)
			}
			if _, ok := resourceData.GetOk("display_apps"); !ok {
				err = resourceData.Set("display_apps", []interface{}{
					map[string]bool{
						"vscode":                 true,
						"vscode_insiders":        false,
						"web_terminal":           true,
						"ssh_helper":             true,
						"port_forwarding_helper": true,
					},
				})
				if err != nil {
					return diag.FromErr(err)
				}
			}

			return updateInitScript(resourceData, i)
		},
		DeleteContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
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
				Description:  `The architecture the agent will run on. Must be one of: "amd64", "armv7", "arm64".`,
				ValidateFunc: validation.StringInSlice([]string{"amd64", "armv7", "arm64"}, false),
			},
			"auth": {
				Type:         schema.TypeString,
				Default:      "token",
				ForceNew:     true,
				Optional:     true,
				Description:  `The authentication type the agent will use. Must be one of: "token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity".`,
				ValidateFunc: validation.StringInSlice([]string{"token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity"}, false),
			},
			"dir": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "The starting directory when a user creates a shell session. Defaults to $HOME.",
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
				Description: "A script to run after the agent starts. The script should exit when it is done to signal that the agent is ready.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"startup_script_timeout": {
				Type:         schema.TypeInt,
				Default:      300,
				ForceNew:     true,
				Optional:     true,
				Description:  "Time in seconds until the agent lifecycle status is marked as timed out during start, this happens when the startup script has not completed (exited) in the given time.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			"shutdown_script": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "A script to run before the agent is stopped. The script should exit when it is done to signal that the workspace can be stopped.",
			},
			"shutdown_script_timeout": {
				Type:         schema.TypeInt,
				Default:      300,
				ForceNew:     true,
				Optional:     true,
				Description:  "Time in seconds until the agent lifecycle status is marked as timed out during shutdown, this happens when the shutdown script has not completed (exited) in the given time.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			"token": {
				ForceNew:    true,
				Sensitive:   true,
				Description: `Set the environment variable "CODER_AGENT_TOKEN" with this token to authenticate an agent.`,
				Type:        schema.TypeString,
				Computed:    true,
			},
			"connection_timeout": {
				Type:         schema.TypeInt,
				Default:      120,
				ForceNew:     true,
				Optional:     true,
				Description:  "Time in seconds until the agent is marked as timed out when a connection with the server cannot be established. A value of zero never marks the agent as timed out.",
				ValidateFunc: validation.IntAtLeast(0),
			},
			"troubleshooting_url": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "A URL to a document with instructions for troubleshooting problems with the agent.",
			},
			"motd_file": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "The path to a file within the workspace containing a message to display to users when they login via SSH. A typical value would be /etc/motd.",
			},
			"login_before_ready": {
				// Note: When this is removed, "startup_script_behavior" should
				// be set to "non-blocking" by default (instead of empty string).
				Type:          schema.TypeBool,
				Default:       true,
				ForceNew:      true,
				Optional:      true,
				Description:   "This option defines whether or not the user can (by default) login to the workspace before it is ready. Ready means that e.g. the startup_script is done and has exited. When enabled, users may see an incomplete workspace when logging in.",
				Deprecated:    "Configure startup_script_behavior instead. This attribute will be removed in a future version of the provider.",
				ConflictsWith: []string{"startup_script_behavior"},
			},
			"startup_script_behavior": {
				// Note: Our default value is "non-blocking" but we do not set
				// it here because we want to be able to differentiate between
				// the user setting this or "login_before_ready". For all
				// intents and purposes, until deprecation, setting
				// "login_before_ready = false" is equivalent to setting
				// "startup_script_behavior = blocking".
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				Description:   "This option sets the behavior of the `startup_script`. When set to \"blocking\", the startup_script must exit before the workspace is ready. When set to \"non-blocking\", the startup_script may run in the background and the workspace will be ready immediately. Default is \"non-blocking\", although \"blocking\" is recommended.",
				ValidateFunc:  validation.StringInSlice([]string{"blocking", "non-blocking"}, false),
				ConflictsWith: []string{"login_before_ready"},
			},
			"metadata": {
				Type:        schema.TypeList,
				Description: "Each \"metadata\" block defines a single item consisting of a key/value pair. This feature is in alpha and may break in future releases.",
				ForceNew:    true,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Description: "The key of this metadata item.",
							ForceNew:    true,
							Required:    true,
						},
						"display_name": {
							Type:        schema.TypeString,
							Description: "The user-facing name of this value.",
							ForceNew:    true,
							Optional:    true,
						},
						"script": {
							Type:        schema.TypeString,
							Description: "The script that retrieves the value of this metadata item.",
							ForceNew:    true,
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"timeout": {
							Type:        schema.TypeInt,
							Description: "The maximum time the command is allowed to run in seconds.",
							ForceNew:    true,
							Optional:    true,
						},
						"interval": {
							Type:        schema.TypeInt,
							Description: "The interval in seconds at which to refresh this metadata item. ",
							ForceNew:    true,
							Required:    true,
						},
						"order": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The order determines the position of agent metadata in the UI/CLI presentation. The lowest order is shown first and parameters with equal order are sorted by key (ascending order).",
						},
					},
				},
			},
			"display_apps": {
				Type:        schema.TypeSet,
				Description: "The list of built-in apps to display in the agent bar.",
				ForceNew:    true,
				Optional:    true,
				MaxItems:    1,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vscode": {
							Type:        schema.TypeBool,
							Description: "Display the VSCode Desktop app in the agent bar.",
							ForceNew:    true,
							Optional:    true,
							Default:     true,
						},
						"vscode_insiders": {
							Type:        schema.TypeBool,
							Description: "Display the VSCode Insiders app in the agent bar.",
							ForceNew:    true,
							Optional:    true,
							Default:     false,
						},
						"web_terminal": {
							Type:        schema.TypeBool,
							Description: "Display the web terminal app in the agent bar.",
							ForceNew:    true,
							Optional:    true,
							Default:     true,
						},
						"port_forwarding_helper": {
							Type:        schema.TypeBool,
							Description: "Display the port-forwarding helper button in the agent bar.",
							ForceNew:    true,
							Optional:    true,
							Default:     true,
						},
						"ssh_helper": {
							Type:        schema.TypeBool,
							Description: "Display the SSH helper button in the agent bar.",
							ForceNew:    true,
							Optional:    true,
							Default:     true,
						},
					},
				},
			},
		},
	}
}

func agentInstanceResource() *schema.Resource {
	return &schema.Resource{
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
