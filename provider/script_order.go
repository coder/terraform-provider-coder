package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// scriptOrderRequiresValues are the values accepted by a rule's "requires"
// attribute. Keep in sync with the consumer in coder/coder's terraform
// provisioner, which validates the same values defensively because
// templates can pin arbitrary provider versions.
var scriptOrderRequiresValues = []string{"success", "completion"}

func scriptOrderDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to declare the execution order of " +
			"`coder_script` resources, and of registry modules that contain them, " +
			"at the template level. Scripts matched by a rule's `run` selectors " +
			"start only after every script matched by that rule's `after` " +
			"selectors has finished; every other script keeps running in " +
			"parallel. This is the recommended way to coordinate workspace " +
			"startup; see the " +
			"[declarative script ordering guide](https://coder.com/docs/admin/templates/startup-coordination/script-ordering) " +
			"for selector syntax, failure semantics, and examples.",
		ReadContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())
			return nil
		},
		Schema: map[string]*schema.Schema{
			"rule": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Description: "An ordering constraint. Add one `rule` block per constraint; multiple blocks merge into a single dependency graph per agent.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"run": {
							Type:        schema.TypeList,
							Required:    true,
							ForceNew:    true,
							MinItems:    1,
							Description: "One or more selectors naming the scripts this rule applies to. Accepts `coder_script.<name>`, `coder_script.<name>[<index>]`, and `module.<name>` (every script in the module, including nested modules), resolved relative to the module where this data source is declared.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"after": {
							Type:        schema.TypeList,
							Required:    true,
							ForceNew:    true,
							MinItems:    1,
							Description: "One or more selectors naming the scripts that every script matched by `run` must wait for. Accepts the same selector syntax as `run`.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"requires": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Default:     "success",
							Description: "Controls what happens when a dependency named in `after` does not succeed. `success` (default) skips the dependent script when a dependency fails, times out, or was itself skipped. `completion` runs the dependent script once its dependencies reach a terminal state, regardless of outcome.",
							ValidateFunc: validation.StringInSlice(scriptOrderRequiresValues, false),
						},
					},
				},
			},
		},
	}
}
