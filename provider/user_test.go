package provider_test

import (
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSSHEd25519PublicKey = `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJeNcdBMtd4Jo9f2W8RZef0ld7Ypye5zTQEf0vUXa/Eq owner123@host456`
	// nolint:gosec  // This key was generated specifically for this purpose.
	testSSHEd25519PrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
	b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
	QyNTUxOQAAACCXjXHQTLXeCaPX9lvEWXn9JXe2Kcnuc00BH9L1F2vxKgAAAJgp3mfQKd5n
	0AAAAAtzc2gtZWQyNTUxOQAAACCXjXHQTLXeCaPX9lvEWXn9JXe2Kcnuc00BH9L1F2vxKg
	AAAEBia7mAQFoLBILlvTJroTkOUomzfcPY9ckpViQOjYFkAZeNcdBMtd4Jo9f2W8RZef0l
	d7Ypye5zTQEf0vUXa/EqAAAAE3ZzY29kZUAzY2Y4MWY5YmM3MmQBAg==
	-----END OPENSSH PRIVATE KEY-----`
)

func TestUserDatasource(t *testing.T) {
	t.Setenv("CODER_USER_ID", "11111111-1111-1111-1111-111111111111")
	t.Setenv("CODER_USER_NAME", "owner123")
	t.Setenv("CODER_USER_AVATAR_URL", "https://example.com/avatar.png")
	t.Setenv("CODER_USER_FULL_NAME", "Mr Owner")
	t.Setenv("CODER_USER_EMAIL", "owner123@example.com")
	t.Setenv("CODER_USER_SSH_PUBLIC_KEY", testSSHEd25519PublicKey)
	t.Setenv("CODER_USER_SSH_PRIVATE_KEY", testSSHEd25519PrivateKey)
	t.Setenv("CODER_USER_GROUPS", `["group1", "group2"]`)

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {}
			data "coder_user" "me" {}
			`,
			Check: func(s *terraform.State) error {
				require.Len(t, s.Modules, 1)
				require.Len(t, s.Modules[0].Resources, 1)
				resource := s.Modules[0].Resources["data.coder_user.me"]
				require.NotNil(t, resource)

				attrs := resource.Primary.Attributes
				assert.Equal(t, "11111111-1111-1111-1111-111111111111", attrs["id"])
				assert.Equal(t, "owner123", attrs["name"])
				assert.Equal(t, "Mr Owner", attrs["full_name"])
				assert.Equal(t, "owner123@example.com", attrs["email"])
				assert.Equal(t, testSSHEd25519PublicKey, attrs["ssh_public_key"])
				assert.Equal(t, testSSHEd25519PrivateKey, attrs["ssh_private_key"])
				assert.Equal(t, `group1`, attrs["groups.0"])
				assert.Equal(t, `group2`, attrs["groups.1"])
				return nil
			},
		}},
	})
}
