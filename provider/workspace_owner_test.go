package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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

func TestWorkspaceOwnerDatasource(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		t.Setenv("CODER_WORKSPACE_OWNER_ID", "11111111-1111-1111-1111-111111111111")
		t.Setenv("CODER_WORKSPACE_OWNER", "owner123")
		t.Setenv("CODER_WORKSPACE_OWNER_NAME", "Mr Owner")
		t.Setenv("CODER_WORKSPACE_OWNER_EMAIL", "owner123@example.com")
		t.Setenv("CODER_WORKSPACE_OWNER_SSH_PUBLIC_KEY", testSSHEd25519PublicKey)
		t.Setenv("CODER_WORKSPACE_OWNER_SSH_PRIVATE_KEY", testSSHEd25519PrivateKey)
		t.Setenv("CODER_WORKSPACE_OWNER_GROUPS", `["group1", "group2"]`)
		t.Setenv("CODER_WORKSPACE_OWNER_SESSION_TOKEN", `supersecret`)
		t.Setenv("CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN", `alsosupersecret`)
		t.Setenv("CODER_WORKSPACE_OWNER_OIDC_ID_TOKEN", `yetanothersecret`)
		t.Setenv("CODER_WORKSPACE_OWNER_LOGIN_TYPE", `github`)
		t.Setenv("CODER_WORKSPACE_OWNER_RBAC_ROLES", `[{"name":"member","org_id":"00000000-0000-0000-0000-000000000000"}]`)

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
			provider "coder" {}
			data "coder_workspace_owner" "me" {}
			`,
				Check: func(s *terraform.State) error {
					require.Len(t, s.Modules, 1)
					require.Len(t, s.Modules[0].Resources, 1)
					resource := s.Modules[0].Resources["data.coder_workspace_owner.me"]
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
					assert.Equal(t, `supersecret`, attrs["session_token"])
					assert.Equal(t, `alsosupersecret`, attrs["oidc_access_token"])
					assert.Equal(t, `yetanothersecret`, attrs["oidc_id_token"])
					assert.Equal(t, `github`, attrs["login_type"])
					assert.Equal(t, `member`, attrs["rbac_roles.0.name"])
					assert.Equal(t, `00000000-0000-0000-0000-000000000000`, attrs["rbac_roles.0.org_id"])
					return nil
				},
			}},
		})
	})

	t.Run("Defaults", func(t *testing.T) {
		for _, v := range []string{
			"CODER_WORKSPACE_OWNER",
			"CODER_WORKSPACE_OWNER_ID",
			"CODER_WORKSPACE_OWNER_EMAIL",
			"CODER_WORKSPACE_OWNER_NAME",
			"CODER_WORKSPACE_OWNER_SESSION_TOKEN",
			"CODER_WORKSPACE_OWNER_GROUPS",
			"CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN",
			"CODER_WORKSPACE_OWNER_OIDC_ID_TOKEN",
			"CODER_WORKSPACE_OWNER_SSH_PUBLIC_KEY",
			"CODER_WORKSPACE_OWNER_SSH_PRIVATE_KEY",
			"CODER_WORKSPACE_OWNER_LOGIN_TYPE",
			"CODER_WORKSPACE_OWNER_RBAC_ROLES",
		} { // https://github.com/golang/go/issues/52817
			t.Setenv(v, "")
			os.Unsetenv(v)
		}

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
			provider "coder" {}
			data "coder_workspace_owner" "me" {}
			`,
				Check: func(s *terraform.State) error {
					require.Len(t, s.Modules, 1)
					require.Len(t, s.Modules[0].Resources, 1)
					resource := s.Modules[0].Resources["data.coder_workspace_owner.me"]
					require.NotNil(t, resource)

					attrs := resource.Primary.Attributes
					assert.NotEmpty(t, attrs["id"])
					assert.Equal(t, "default", attrs["name"])
					assert.Equal(t, "default", attrs["full_name"])
					assert.Equal(t, "default@example.com", attrs["email"])
					assert.Empty(t, attrs["ssh_public_key"])
					assert.Empty(t, attrs["ssh_private_key"])
					assert.Empty(t, attrs["groups.0"])
					assert.Empty(t, attrs["session_token"])
					assert.Empty(t, attrs["oidc_access_token"])
					assert.Empty(t, attrs["oidc_id_token"])
					assert.Empty(t, attrs["login_type"])
					assert.Empty(t, attrs["rbac_roles.0"])
					return nil
				},
			}},
		})
	})
}
