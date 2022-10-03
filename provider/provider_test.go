package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestProvider(t *testing.T) {
	t.Parallel()
	tfProvider := provider.New()
	err := tfProvider.InternalValidate()
	require.NoError(t, err)

	t.Run("Environment", func(t *testing.T) {
		t.Setenv("CODER_AGENT_URL", "https://dev.coder.com:12345")
		data := schema.TestResourceDataRaw(t, tfProvider.Schema, map[string]interface{}{})
		rawConfig, diags := tfProvider.ConfigureContextFunc(context.Background(), data)
		require.Nil(t, diags)
		config, ok := rawConfig.(provider.Config)
		require.True(t, ok)
		require.Equal(t, "https://dev.coder.com:12345", config.URL.String())
	})

	for _, tc := range []struct {
		InputHost string
		InputURL  string
		OutputURL string
	}{{
		InputHost: "",
		InputURL:  "https://coder.com",
		OutputURL: "https://coder.com",
	}, {
		InputHost: "bananas:443",
		InputURL:  "https://wow.com:12345",
		OutputURL: "https://bananas:443",
	}, {
		InputHost: "host.docker.internal",
		InputURL:  "http://localhost:8080",
		OutputURL: "http://host.docker.internal:8080",
	}} {
		tc := tc
		t.Run(tc.OutputURL, func(t *testing.T) {
			t.Parallel()
			data := schema.TestResourceDataRaw(t, tfProvider.Schema, map[string]interface{}{
				"url":  tc.InputURL,
				"host": tc.InputHost,
			})
			rawConfig, diags := tfProvider.ConfigureContextFunc(context.Background(), data)
			require.Nil(t, diags)
			config, ok := rawConfig.(provider.Config)
			require.True(t, ok)
			require.Equal(t, tc.OutputURL, config.URL.String())
		})
	}
}

func echoProvider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},
		DataSourcesMap: map[string]*schema.Resource{
			"echo": {
				Schema: map[string]*schema.Schema{
					"value": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
	}
}
