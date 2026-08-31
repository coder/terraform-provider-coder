package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func scriptOrderDataSource() *schema.Resource {
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
		SchemaVersion: 1,

		Description: "Use this data source to declare ordering constraints between existing lifecycle-triggered `coder_script` executions. It does not cause scripts to run. Coder resolves the Terraform address selectors after planning.",
		ReadContext: func(_ context.Context, resourceData *schema.ResourceData, _ any) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())
			return nil
		},
		Schema: map[string]*schema.Schema{
			"rule": {
				Type:        schema.TypeList,
				Description: "One or more ordering rules. Every script selected by `run` waits for every script selected by `after` according to `requires`.",
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"run":   selectorSchema("Terraform address selectors for the dependent scripts."),
						"after": selectorSchema("Terraform address selectors for the prerequisite scripts."),
						"requires": {
							Type:         schema.TypeString,
							Description:  "The outcome required from prerequisite scripts. The default is `success`, which causes dependent scripts to be skipped when a prerequisite does not succeed. `completion` runs dependent scripts after all prerequisites reach a terminal outcome.",
							Optional:     true,
							Default:      "success",
							ValidateFunc: validation.StringInSlice([]string{"success", "completion"}, false),
						},
					},
				},
			},
		},
	}
}
