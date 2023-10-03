package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// externalAuthDataSource returns a schema for an external authentication data source.
func externalAuthDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to require users to authenticate with an external provider prior to workspace creation. This can be used to pre-authenticate external services in a workspace.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			id, ok := rd.Get("id").(string)
			if !ok || id == "" {
				return diag.Errorf("id is required")
			}
			rd.SetId(id)

			accessToken := os.Getenv(ExternalAuthAccessTokenEnvironmentVariable(id))
			rd.Set("access_token", accessToken)
			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of a configured external auth provider set up in your Coder deployment.",
				Required:    true,
			},
			"access_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access token returned by the external auth provider. This can be used to pre-authenticate command-line tools.",
			},
		},
	}
}

func ExternalAuthAccessTokenEnvironmentVariable(id string) string {
	return fmt.Sprintf("CODER_EXTERNAL_AUTH_ACCESS_TOKEN_%s", id)
}
