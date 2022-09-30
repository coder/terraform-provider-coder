package provider

import (
	"context"
	"net/url"
	"reflect"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/xerrors"
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
				// The "CODER_AGENT_URL" environment variable is used by default
				// as the Access URL when generating scripts.
				DefaultFunc: schema.EnvDefaultFunc("CODER_AGENT_URL", "https://mydeployment.coder.com"),
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
				return nil, diag.Errorf("CODER_AGENT_URL must not be empty; got %q", rawURL)
			}
			parsed, err := url.Parse(resourceData.Get("url").(string))
			if err != nil {
				return nil, diag.FromErr(err)
			}
			rawHost, ok := resourceData.Get("host").(string)
			if ok && rawHost != "" {
				rawPort := parsed.Port()
				if rawPort != "" && !strings.Contains(rawHost, ":") {
					rawHost += ":" + rawPort
				}
				parsed.Host = rawHost
			}
			return config{
				URL: parsed,
			}, nil
		},
		DataSourcesMap: map[string]*schema.Resource{
			"coder_workspace":   workspaceDataSource(),
			"coder_provisioner": provisionerDataSource(),
			"coder_parameter":   parameterDataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"coder_agent":          agentResource(),
			"coder_agent_instance": agentInstanceResource(),
			"coder_app":            appResource(),
			"coder_metadata":       metadataResource(),
		},
	}
}

// populateIsNull reads the raw plan for a coder_metadata resource being created,
// figures out which items have null "value"s, and augments them by setting the
// "is_null" field to true. This ugly hack is necessary because terraform-plugin-sdk
// is designed around a old version of Terraform that didn't support nullable fields,
// and it doesn't correctly propagate null values for primitive types.
// Returns an interface{} representing the new value of the "item" field, or an error.
func populateIsNull(resourceData *schema.ResourceData) (result interface{}, err error) {
	// The cty package reports type mismatches by panicking
	defer func() {
		if r := recover(); r != nil {
			err = xerrors.Errorf("panic while handling coder_metadata: %#v", r)
		}
	}()

	rawPlan := resourceData.GetRawPlan()
	items := rawPlan.GetAttr("item").AsValueSlice()

	var resultItems []interface{}
	itemKeys := map[string]struct{}{}
	for _, item := range items {
		key := valueAsString(item.GetAttr("key"))
		_, exists := itemKeys[key]
		if exists {
			return nil, xerrors.Errorf("duplicate metadata key %q", key)
		}
		itemKeys[key] = struct{}{}
		resultItem := map[string]interface{}{
			"key":       key,
			"value":     valueAsString(item.GetAttr("value")),
			"sensitive": valueAsBool(item.GetAttr("sensitive")),
		}
		if item.GetAttr("value").IsNull() {
			resultItem["is_null"] = true
		}
		resultItems = append(resultItems, resultItem)
	}

	return resultItems, nil
}

// valueAsString takes a cty.Value that may be a string or null, and converts it to a Go string,
// which will be empty if the input value was null.
// or a nil interface{}
func valueAsString(value cty.Value) string {
	if value.IsNull() {
		return ""
	}
	return value.AsString()
}

// valueAsString takes a cty.Value that may be a boolean or null, and converts it to either a Go bool
// or a nil interface{}
func valueAsBool(value cty.Value) interface{} {
	if value.IsNull() {
		return nil
	}
	return value.True()
}

// errorAsDiagnostic transforms a Go error to a diag.Diagnostics object representing a fatal error.
func errorAsDiagnostics(err error) diag.Diagnostics {
	return []diag.Diagnostic{{
		Severity: diag.Error,
		Summary:  err.Error(),
	}}
}
