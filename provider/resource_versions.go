package provider

// ResourceVersions maps each resource and data source to its minimum required Coder version.
// IMPORTANT: When adding a new resource or data source, you MUST add an entry here!
// This ensures proper documentation of version requirements for users.
var ResourceVersions = map[string]string{
	// Resources
	"coder_agent":          "v2.18.0", // Base requirement for terraform-provider-coder v2.0+
	"coder_agent_instance": "v2.18.0",
	"coder_ai_task":        "v2.24.0", // AI features introduced in v2.24.0
	"coder_app":            "v2.18.0",
	"coder_devcontainer":   "v2.21.0", // Devcontainer support added in v2.21.0
	"coder_env":            "v2.18.0",
	"coder_metadata":       "v2.18.0",
	"coder_script":         "v2.18.0",

	// Data Sources
	"coder_external_auth":    "v2.18.0",
	"coder_parameter":        "v2.18.0",
	"coder_provisioner":      "v2.18.0",
	"coder_workspace":        "v2.18.0",
	"coder_workspace_owner":  "v2.18.0",
	"coder_workspace_preset": "v2.18.0",
	"coder_workspace_tags":   "v2.18.0",
}

// AttributeVersions maps specific resource attributes to their minimum required Coder version.
// Use this for attributes that were added after the resource itself.
var AttributeVersions = map[string]map[string]string{
	"coder_app": {
		"hidden": "v2.16.0", // Hidden attribute added in v2.16.0
	},
}
