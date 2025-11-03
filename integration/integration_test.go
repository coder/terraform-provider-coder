package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"golang.org/x/mod/semver"
)

// TestIntegration performs an integration test against an ephemeral Coder deployment.
// For each directory containing a `main.tf` under `/integration`, performs the following:
//   - Pushes the template to a temporary Coder instance running in Docker
//   - Creates a workspace from the template. Templates here are expected to create a
//     local_file resource containing JSON that can be marshalled as a map[string]string
//   - Fetches the content of the JSON file created and compares it against the expected output.
//
// NOTE: all interfaces to this Coder deployment are performed without github.com/coder/coder/v2/codersdk
// in order to avoid a circular dependency.
func TestIntegration(t *testing.T) {
	if os.Getenv("TF_ACC") == "1" {
		t.Skip("Skipping integration tests during tf acceptance tests")
	}

	coderImg := os.Getenv("CODER_IMAGE")
	if coderImg == "" {
		coderImg = "ghcr.io/coder/coder-preview"
	}

	coderVersion := os.Getenv("CODER_VERSION")
	if coderVersion == "" {
		coderVersion = "latest"
	}

	timeoutStr := os.Getenv("TIMEOUT_MINS")
	if timeoutStr == "" {
		timeoutStr = "10"
	}
	timeoutMins, err := strconv.Atoi(timeoutStr)
	require.NoError(t, err, "invalid value specified for timeout")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMins)*time.Minute)
	t.Cleanup(cancel)
	ctrID := setup(ctx, t, t.Name(), coderImg, coderVersion)

	for _, tt := range []struct {
		// Name of the folder under `integration/` containing a test template
		name string
		// Minimum coder version for which to run this test
		minVersion string
		// map of string to regex to be passed to assertOutput()
		expectedOutput map[string]string
	}{
		{
			name:       "test-data-source",
			minVersion: "v0.0.0",
			expectedOutput: map[string]string{
				"provisioner.arch":                runtime.GOARCH,
				"provisioner.id":                  `[a-zA-Z0-9-]+`,
				"provisioner.os":                  runtime.GOOS,
				"workspace.access_port":           `\d+`,
				"workspace.access_url":            `https?://\D+:\d+`,
				"workspace.id":                    `[a-zA-z0-9-]+`,
				"workspace.name":                  `test-data-source`,
				"workspace.start_count":           `1`,
				"workspace.template_id":           `[a-zA-Z0-9-]+`,
				"workspace.template_name":         `test-data-source`,
				"workspace.template_version":      `.+`,
				"workspace.transition":            `start`,
				"workspace_parameter.name":        `param`,
				"workspace_parameter.description": `param description`,
				// TODO (sasswart): the cli doesn't support presets yet.
				// once it does, the value for workspace_parameter.value
				// will be the preset value.
				"workspace_parameter.value":                                 `param value`,
				"workspace_parameter.icon":                                  `param icon`,
				"workspace_preset.name":                                     `preset`,
				"workspace_preset.description":                              `preset description`,
				"workspace_preset.icon":                                     `preset icon`,
				"workspace_preset.default":                                  `true`,
				"workspace_preset.parameters.param":                         `preset param value`,
				"workspace_preset.prebuilds.instances":                      `1`,
				"workspace_preset.prebuilds.expiration_policy.ttl":          `86400`,
				"workspace_preset.prebuilds.scheduling.timezone":            `UTC`,
				"workspace_preset.prebuilds.scheduling.schedule0.cron":      `\* 8-18 \* \* 1-5`,
				"workspace_preset.prebuilds.scheduling.schedule0.instances": `3`,
				"workspace_preset.prebuilds.scheduling.schedule1.cron":      `\* 8-14 \* \* 6`,
				"workspace_preset.prebuilds.scheduling.schedule1.instances": `1`,
			},
		},
		{
			name:       "workspace-owner",
			minVersion: "v2.12.0",
			expectedOutput: map[string]string{
				"provisioner.arch":                  runtime.GOARCH,
				"provisioner.id":                    `[a-zA-Z0-9-]+`,
				"provisioner.os":                    runtime.GOOS,
				"workspace.access_port":             `\d+`,
				"workspace.access_url":              `https?://\D+:\d+`,
				"workspace.id":                      `[a-zA-z0-9-]+`,
				"workspace.name":                    ``,
				"workspace.start_count":             `1`,
				"workspace.template_id":             `[a-zA-Z0-9-]+`,
				"workspace.template_name":           `workspace-owner`,
				"workspace.template_version":        `.+`,
				"workspace.transition":              `start`,
				"workspace_owner.email":             `testing@coder\.com`,
				"workspace_owner.full_name":         `default`,
				"workspace_owner.groups":            `\[(\"Everyone\")?\]`,
				"workspace_owner.id":                `[a-zA-Z0-9-]+`,
				"workspace_owner.name":              `testing`,
				"workspace_owner.oidc_access_token": `^$`, // TODO: test OIDC integration
				"workspace_owner.session_token":     `.+`,
				"workspace_owner.ssh_private_key":   `(?s)^.+?BEGIN OPENSSH PRIVATE KEY.+?END OPENSSH PRIVATE KEY.+?$`,
				"workspace_owner.ssh_public_key":    `(?s)^ssh-ed25519.+$`,
				"workspace_owner.login_type":        ``,
				"workspace_owner.rbac_roles":        ``,
			},
		},
		{
			name:       "workspace-owner-filled",
			minVersion: "v2.18.0",
			expectedOutput: map[string]string{
				"provisioner.arch":                  runtime.GOARCH,
				"provisioner.id":                    `[a-zA-Z0-9-]+`,
				"provisioner.os":                    runtime.GOOS,
				"workspace.access_port":             `\d+`,
				"workspace.access_url":              `https?://\D+:\d+`,
				"workspace.id":                      `[a-zA-z0-9-]+`,
				"workspace.name":                    ``,
				"workspace.start_count":             `1`,
				"workspace.template_id":             `[a-zA-Z0-9-]+`,
				"workspace.template_name":           `workspace-owner`,
				"workspace.template_version":        `.+`,
				"workspace.transition":              `start`,
				"workspace_owner.email":             `testing@coder\.com`,
				"workspace_owner.full_name":         `default`,
				"workspace_owner.groups":            `\[(\"Everyone\")?\]`,
				"workspace_owner.id":                `[a-zA-Z0-9-]+`,
				"workspace_owner.name":              `testing`,
				"workspace_owner.oidc_access_token": `^$`, // TODO: test OIDC integration
				"workspace_owner.session_token":     `.+`,
				"workspace_owner.ssh_private_key":   `(?s)^.+?BEGIN OPENSSH PRIVATE KEY.+?END OPENSSH PRIVATE KEY.+?$`,
				"workspace_owner.ssh_public_key":    `(?s)^ssh-ed25519.+$`,
				"workspace_owner.login_type":        `password`,
				"workspace_owner.rbac_roles":        ``,
			},
		},
		{
			name:       "workspace-owner-rbac-roles",
			minVersion: "v2.21.0", // anticipated version, update as required
			expectedOutput: map[string]string{
				"provisioner.arch":                  runtime.GOARCH,
				"provisioner.id":                    `[a-zA-Z0-9-]+`,
				"provisioner.os":                    runtime.GOOS,
				"workspace.access_port":             `\d+`,
				"workspace.access_url":              `https?://\D+:\d+`,
				"workspace.id":                      `[a-zA-z0-9-]+`,
				"workspace.name":                    ``,
				"workspace.start_count":             `1`,
				"workspace.template_id":             `[a-zA-Z0-9-]+`,
				"workspace.template_name":           `workspace-owner`,
				"workspace.template_version":        `.+`,
				"workspace.transition":              `start`,
				"workspace_owner.email":             `testing@coder\.com`,
				"workspace_owner.full_name":         `default`,
				"workspace_owner.groups":            `\[(\"Everyone\")?\]`,
				"workspace_owner.id":                `[a-zA-Z0-9-]+`,
				"workspace_owner.name":              `testing`,
				"workspace_owner.oidc_access_token": `^$`, // TODO: test OIDC integration
				"workspace_owner.session_token":     `.+`,
				"workspace_owner.ssh_private_key":   `(?s)^.+?BEGIN OPENSSH PRIVATE KEY.+?END OPENSSH PRIVATE KEY.+?$`,
				"workspace_owner.ssh_public_key":    `(?s)^ssh-ed25519.+$`,
				"workspace_owner.login_type":        `password`,
				// org_id will either be a uuid or an empty string for site wide roles.
				"workspace_owner.rbac_roles": `(?is)\[(\{"name":"[a-z0-9-:]+","org_id":"[a-f0-9-]*"\},?)+\]`,
			},
		},
		{
			name:       "coder-app-open-in",
			minVersion: "v2.19.0",
			expectedOutput: map[string]string{
				"coder_app.tab.open_in":       "tab",
				"coder_app.defaulted.open_in": "slim-window",
			},
		},
		{
			name:       "coder-app-hidden",
			minVersion: "v0.0.0",
			expectedOutput: map[string]string{
				"coder_app.hidden.hidden":    "true",
				"coder_app.visible.hidden":   "false",
				"coder_app.defaulted.hidden": "false",
			},
		},
		{
			name:       "coder-ai-task",
			minVersion: "v2.26.0",
			expectedOutput: map[string]string{
				"ai_task.id":             `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
				"ai_task.prompt":         "",
				"ai_task.app_id":         `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
				"ai_task.enabled":        "false",
				"app.id":                 `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
				"ai_task_prompt.id":      `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
				"ai_task_prompt.value":   "",
				"ai_task_prompt.enabled": "false",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if coderVersion != "latest" && semver.Compare(coderVersion, tt.minVersion) < 0 {
				t.Skipf("skipping due to CODER_VERSION %q < minVersion %q", coderVersion, tt.minVersion)
			}

			// Given: we have an existing Coder deployment running locally
			// Import named template

			// NOTE: Template create command was deprecated after this version
			// ref: https://github.com/coder/coder/pull/11390
			templateCreateCmd := "push"
			if semver.Compare(coderVersion, "v2.7.0") < 1 {
				t.Logf("using now-deprecated templates create command for older coder version")
				templateCreateCmd = "create"
			}
			_, rc := execContainer(ctx, t, ctrID, fmt.Sprintf(`coder templates %s %s --directory /src/integration/%s --var output_path=/tmp/%s.json --yes`, templateCreateCmd, tt.name, tt.name, tt.name))
			require.Equal(t, 0, rc)

			// Check if parameters.yaml exists
			_, rc = execContainer(ctx, t, ctrID, fmt.Sprintf(`stat /src/integration/%s/parameters.yaml 2>/dev/null > /dev/null`, tt.name))
			hasParameters := rc == 0
			var includeParameters string
			if hasParameters {
				// If it exists, include it in the create command
				includeParameters = fmt.Sprintf(`--rich-parameter-file /src/integration/%s/parameters.yaml`, tt.name)
			}

			// Create a workspace
			_, rc = execContainer(ctx, t, ctrID, fmt.Sprintf(`coder create %s -t %s %s --yes`, tt.name, tt.name, includeParameters))
			require.Equal(t, 0, rc)
			// Fetch the output created by the template
			out, rc := execContainer(ctx, t, ctrID, fmt.Sprintf(`cat /tmp/%s.json`, tt.name))
			require.Equal(t, 0, rc)
			actual := make(map[string]string)
			require.NoError(t, json.NewDecoder(strings.NewReader(out)).Decode(&actual))
			assertOutput(t, tt.expectedOutput, actual)
		})
	}
}

func setup(ctx context.Context, t *testing.T, name, coderImg, coderVersion string) string {
	var (
		// For this test to work, we pass in a custom terraformrc to use
		// the locally built version of the provider.
		testTerraformrc = `provider_installation {
		dev_overrides {
		  "coder/coder" = "/src"
		}
		  direct{}
	  }`
		localURL = "http://localhost:3000"
	)

	t.Logf("using coder image %s:%s", coderImg, coderVersion)

	// Ensure the binary is built
	binPath, err := filepath.Abs("../terraform-provider-coder")
	require.NoError(t, err)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Fatalf("not found: %q - please build the provider first", binPath)
	}
	tmpDir := t.TempDir()
	// Create a terraformrc to point to our freshly built provider!
	tfrcPath := filepath.Join(tmpDir, "integration.tfrc")
	err = os.WriteFile(tfrcPath, []byte(testTerraformrc), 0o644)
	require.NoError(t, err, "write terraformrc to tempdir")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err, "init docker client")

	srcPath, err := filepath.Abs("..")
	require.NoError(t, err, "get abs path of parent")
	t.Logf("src path is %s\n", srcPath)

	// Ensure the image is available locally.
	refStr := coderImg + ":" + coderVersion
	ensureImage(ctx, t, cli, refStr)

	// Stand up a temporary Coder instance
	ctr, err := cli.ContainerCreate(ctx, &container.Config{
		Image: refStr,
		Env: []string{
			"CODER_ACCESS_URL=" + localURL,             // Set explicitly to avoid creating try.coder.app URLs.
			"CODER_IN_MEMORY=true",                     // We don't necessarily care about real persistence here.
			"CODER_TELEMETRY_ENABLE=false",             // Avoid creating noise.
			"CODER_VERBOSE=TRUE",                       // Debug logging.
			"TF_CLI_CONFIG_FILE=/tmp/integration.tfrc", // Our custom tfrc from above.
			"TF_LOG=DEBUG",                             // Debug logging in Terraform provider
		},
		Labels: map[string]string{},
	}, &container.HostConfig{
		Binds: []string{
			tfrcPath + ":/tmp/integration.tfrc", // Custom tfrc from above.
			srcPath + ":/src",                   // Bind-mount in the repo with the built binary and templates.
		},
	}, nil, nil, "terraform-provider-coder-integration-"+name)
	require.NoError(t, err, "create test deployment")

	t.Logf("created container %s\n", ctr.ID)
	var cleanupOnce sync.Once
	removeContainer := func() {
		t.Logf("stopping container %s\n", ctr.ID)
		_ = cli.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{
			Force: true,
		})
	}
	// Ensure the container is cleaned up if you press Ctrl+C.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cleanupOnce.Do(removeContainer)
		os.Exit(1)
	}()

	t.Cleanup(func() { // Make sure we clean up after ourselves.
		cleanupOnce.Do(removeContainer)
	})

	err = cli.ContainerStart(ctx, ctr.ID, container.StartOptions{})
	require.NoError(t, err, "start container")
	t.Logf("started container %s\n", ctr.ID)

	// nolint:gosec // For testing only.
	var (
		testEmail    = "testing@coder.com"
		testPassword = "InsecurePassw0rd!"
		testUsername = "testing"
	)

	// Wait for container to come up
	require.Eventually(t, func() bool {
		_, rc := execContainer(ctx, t, ctr.ID, fmt.Sprintf(`curl -s --fail %s/api/v2/buildinfo`, localURL))
		if rc == 0 {
			return true
		}
		t.Logf("not ready yet...")
		return false
	}, 10*time.Second, time.Second, "coder failed to become ready in time")

	// Perform first time setup
	_, rc := execContainer(ctx, t, ctr.ID, fmt.Sprintf(`coder login %s --first-user-email=%q --first-user-password=%q --first-user-trial=false --first-user-username=%q`, localURL, testEmail, testPassword, testUsername))
	require.Equal(t, 0, rc, "failed to perform first-time setup")
	return ctr.ID
}

func ensureImage(ctx context.Context, t *testing.T, cli *client.Client, ref string) {
	t.Helper()

	t.Logf("ensuring image %q", ref)
	images, err := cli.ImageList(ctx, image.ListOptions{})
	require.NoError(t, err, "list images")
	for _, img := range images {
		if slices.Contains(img.RepoTags, ref) {
			t.Logf("image %q found locally, not pulling", ref)
			return
		}
	}
	t.Logf("image %s not found locally, attempting to pull", ref)
	resp, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	require.NoError(t, err)
	_, err = io.ReadAll(resp)
	require.NoError(t, err)
}

// execContainer executes the given command in the given container and returns
// the output and the exit code of the command.
func execContainer(ctx context.Context, t *testing.T, containerID, command string) (string, int) {
	t.Helper()
	t.Logf("exec container cmd: %q", command)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err, "connect to docker")
	defer cli.Close()
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", command},
	}
	ex, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	require.NoError(t, err, "create container exec")
	resp, err := cli.ContainerExecAttach(ctx, ex.ID, types.ExecStartCheck{})
	require.NoError(t, err, "attach to container exec")
	defer resp.Close()
	var buf bytes.Buffer
	_, err = stdcopy.StdCopy(&buf, &buf, resp.Reader)
	require.NoError(t, err, "read stdout")
	out := buf.String()
	t.Log("exec container output:\n" + out)
	execResp, err := cli.ContainerExecInspect(ctx, ex.ID)
	require.NoError(t, err, "get exec exit code")
	return out, execResp.ExitCode
}

// assertOutput asserts that, for each key-value pair in expected:
// 1. actual[k] as a regex matches expected[k], and
// 2. the set of keys of expected are not a subset of actual.
func assertOutput(t *testing.T, expected, actual map[string]string) {
	t.Helper()

	for expectedKey, expectedValExpr := range expected {
		actualVal := actual[expectedKey]
		assert.Regexp(t, expectedValExpr, actualVal, "output key %q does not have expected value", expectedKey)
	}
	for actualKey := range actual {
		_, ok := expected[actualKey]
		assert.True(t, ok, "unexpected field in actual %q=%q", actualKey, actual[actualKey])
	}
}
