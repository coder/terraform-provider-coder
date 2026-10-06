package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Coder's provisioner accepts the same values. The provider and Coder
// share no Go types, so the plan JSON is the contract between them.
const (
	scriptOrderRequiresSuccess    = "success"
	scriptOrderRequiresCompletion = "completion"

	scriptOrderPhaseStart = "start"
	scriptOrderPhaseStop  = "stop"
)

func scriptOrderDataSource() *schema.Resource {
	// Selector syntax is not checked here. Coder resolves selectors
	// against the whole plan and reports unknown or unsupported ones at
	// template import. A stricter check here could reject selectors that
	// Coder accepts.
	selectorSchema := func(description string) *schema.Schema {
		return &schema.Schema{
			Type:        schema.TypeList,
			Description: description,
			Required:    true,
			MinItems:    1,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		}
	}

	return &schema.Resource{
		SchemaVersion: 0,

		Description: "Use this data source to declare ordering constraints between `coder_script` " +
			"resources on the same agent. It does not run scripts. Every script selected by " +
			"`run` waits for every script selected by `after`.\n\n" +
			"Selectors are Terraform address strings (`coder_script.name`, " +
			"`coder_script.name[0]`, `coder_script.name[\"key\"]`, `module.name`), resolved " +
			"by Coder relative to the module that declares the data source. Terraform and " +
			"`terraform validate` do not check them; Coder reports unknown or unsupported " +
			"selectors when the template is imported. All values must be known at plan time, " +
			"and the data source needs no `depends_on`.\n\n" +
			"-> This data source is only available in Coder v2.39 and later. Older versions " +
			"ignore it and run scripts concurrently.",
		ReadContext: func(_ context.Context, rd *schema.ResourceData, _ any) diag.Diagnostics {
			rd.SetId(uuid.NewString())
			return nil
		},
		Schema: map[string]*schema.Schema{
			"rule": {
				Type:        schema.TypeList,
				Description: "One or more ordering rules. Rules from every `coder_script_order` in the template are merged into one graph per agent.",
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"run":   selectorSchema("Selectors for the scripts that wait for every `after` script."),
						"after": selectorSchema("Selectors for the scripts that must finish before any `run` script starts."),
						"requires": {
							Type: schema.TypeString,
							Description: "The outcome required from every `after` script before a `run` script starts. " +
								"`success` (the default) skips the `run` scripts if any `after` script fails, times out or is skipped. " +
								"`completion` runs them once every `after` script has finished, whatever the outcome.",
							Optional: true,
							Default:  scriptOrderRequiresSuccess,
							ValidateFunc: validation.StringInSlice([]string{
								scriptOrderRequiresSuccess,
								scriptOrderRequiresCompletion,
							}, false),
						},
						"phase": {
							Type: schema.TypeString,
							Description: "Which lifecycle the rule applies to: `start` or `stop`. " +
								"When omitted, Coder infers the phase from any `coder_script` selector in the rule " +
								"and filters `module` selectors to that phase with a warning. " +
								"If the rule has only `module` selectors and they expand to both start and stop " +
								"scripts, Coder cannot infer the phase and rejects the rule at template import, " +
								"so set `phase` explicitly in that case.",
							Optional: true,
							// No default on purpose. Coder infers the phase when
							// this is unset, warns when inference filtered a
							// module selector, and rejects a module-only rule
							// that spans both phases. A default would hide all
							// three.
							ValidateFunc: validation.StringInSlice([]string{
								scriptOrderPhaseStart,
								scriptOrderPhaseStop,
							}, false),
						},
					},
				},
			},
		},
	}
}
