package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/provider/helpers"
)

// gitAuthDataSource returns a schema for a Git authentication data source.
func gitAuthDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		DeprecationMessage: "Use the `coder_external_auth` data source instead.",
		Description:        "**Deprecated**: use the `coder_external_auth` data source instead. Use this data source to require users to authenticate with a Git provider prior to workspace creation. This can be used to perform an authenticated `git clone` in startup scripts.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			rawID, ok := rd.GetOk("id")
			if !ok {
				return diag.Errorf("id is required")
			}
			id, ok := rawID.(string)
			if !ok {
				return diag.Errorf("unexpected type %q for id", rawID)
			}
			rd.SetId(id)

			accessToken := helpers.OptionalEnv(GitAuthAccessTokenEnvironmentVariable(id))
			rd.Set("access_token", accessToken)

			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of a configured git auth provider set up in your Coder deployment.",
			},
			"access_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access token returned by the git authentication provider. This can be used to pre-authenticate command-line tools.",
			},
		},
	}
}

func GitAuthAccessTokenEnvironmentVariable(id string) string {
	return fmt.Sprintf("CODER_GIT_AUTH_ACCESS_TOKEN_%s", id)
}
