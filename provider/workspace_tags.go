package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const TerraformWorkDirEnv = "CODER_TF_WORK_DIR"

type WorkspaceTags struct {
	Tag []Tag
}

type Tag struct {
	Name  string
	Value string
}

func workspaceTagDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to configure workspace tags to select provisioners.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())
			return nil
		},
		Schema: map[string]*schema.Schema{
			"tag": {
				Type:        schema.TypeList,
				Description: `Each "tag" block defines a workspace tag.`,
				ForceNew:    true,
				Optional:    true,
				MaxItems:    64,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the tag.",
							ForceNew:    true,
							Required:    true,
						},
						"value": {
							Type:        schema.TypeString,
							Description: "The value of the tag.",
							ForceNew:    true,
							Required:    true,
						},
					},
				},
			},
		},
	}
}
