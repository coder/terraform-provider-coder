package provider_test

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestDecode(t *testing.T) {
	const displayName = "Display Name"

	aMap := map[string]interface{}{
		"name":         "Parameter Name",
		"type":         "number",
		"display_name": displayName,
		"validation": []map[string]interface{}{
			{
				"min":          nil,
				"min_disabled": false,
				"max":          5,
				"max_disabled": true,
			},
		},
	}

	var param provider.Parameter
	err := mapstructure.Decode(aMap, &param)
	require.NoError(t, err)
	assert.Equal(t, displayName, param.DisplayName)
	assert.Equal(t, 5, param.Validation[0].Max)
	assert.True(t, param.Validation[0].MaxDisabled)
	assert.Equal(t, 0, param.Validation[0].Min)
	assert.False(t, param.Validation[0].MinDisabled)
}
