package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	if os.Getenv("TF_ACC") == "1" {
		t.Skip("Skipping integration tests during tf acceptance tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	ctrID := setup(ctx, t)

	t.Run("test-data-sources", func(t *testing.T) {
		// Import an example template
		_, rc := execContainer(ctx, t, ctrID, `coder templates push test-data-source --directory /src/integration/test-data-source/ --var output_path=/tmp/test-data-sources.json --yes`)
		require.Equal(t, 0, rc)
		// Create a workspace
		_, rc = execContainer(ctx, t, ctrID, `coder create test-data-source -t test-data-source --yes`)
		require.Equal(t, 0, rc)
		// Fetch the output created by the template
		out, rc := execContainer(ctx, t, ctrID, `cat /tmp/test-data-sources.json`)
		require.Equal(t, 0, rc)
		m := make(map[string]string)
		require.NoError(t, json.NewDecoder(strings.NewReader(out)).Decode(&m))
		assert.Equal(t, runtime.GOARCH, m["provisioner.arch"])
		assert.NotEmpty(t, m["provisioner.id"])
		assert.Equal(t, runtime.GOOS, m["provisioner.os"])
		assert.NotEmpty(t, m["workspace.access_port"])
		assert.NotEmpty(t, m["workspace.access_url"])
		assert.NotEmpty(t, m["workspace.id"])
		assert.Equal(t, "test-data-source", m["workspace.name"])
		assert.Equal(t, "testing", m["workspace.owner"])
		assert.Equal(t, "testing@coder.com", m["workspace.owner_email"])
		assert.NotEmpty(t, m["workspace.owner_id"])
		assert.Equal(t, "default", m["workspace.owner_name"])
		// assert.NotEmpty(t, m["workspace.owner_oidc_access_token"]) // TODO: need a test OIDC integration
		assert.NotEmpty(t, m["workspace.owner_session_token"])
		assert.Equal(t, "1", m["workspace.start_count"])
		assert.NotEmpty(t, m["workspace.template_id"])
		assert.Equal(t, "test-data-source", m["workspace.template_name"])
		assert.NotEmpty(t, m["workspace.template_version"])
		assert.Equal(t, "start", m["workspace.transition"])
		assert.Equal(t, "testing@coder.com", m["workspace_owner.email"])
		assert.Equal(t, "default", m["workspace_owner.full_name"])
		assert.NotEmpty(t, m["workspace_owner.groups"])
		assert.NotEmpty(t, m["workspace_owner.id"])
		assert.Equal(t, "testing", m["workspace_owner.name"])
		// assert.NotEmpty(t, m["workspace_owner.oidc_access_token"]) // TODO: test OIDC integration
		assert.NotEmpty(t, m["workspace_owner.session_token"])
		assert.NotEmpty(t, m["workspace_owner.ssh_private_key"])
		assert.NotEmpty(t, m["workspace_owner.ssh_public_key"])
	})
}

func setup(ctx context.Context, t *testing.T) string {
	coderImg := os.Getenv("CODER_IMAGE")
	if coderImg == "" {
		coderImg = "ghcr.io/coder/coder"
	}

	coderVersion := os.Getenv("CODER_VERSION")
	if coderVersion == "" {
		coderVersion = "latest"
	}

	t.Logf("using coder image %s:%s", coderImg, coderVersion)

	// Ensure the binary is built
	binPath, err := filepath.Abs("../terraform-provider-coder")
	require.NoError(t, err)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Fatalf("not found: %q - please build the provider first", binPath)
	}
	tmpDir := t.TempDir()
	tfrcPath := filepath.Join(tmpDir, "integration.tfrc")
	tfrc := `provider_installation {
  dev_overrides {
    "coder/coder" = "/src"
  }
	direct{}
}`
	err = os.WriteFile(tfrcPath, []byte(tfrc), 0o644)
	require.NoError(t, err, "write terraformrc to tempdir")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err, "init docker client")

	p := randomPort(t)
	t.Logf("random port is %d\n", p)

	srcPath, err := filepath.Abs("..")
	require.NoError(t, err, "get abs path of parent")
	t.Logf("src path is %s\n", srcPath)

	ctr, err := cli.ContainerCreate(ctx, &container.Config{
		Image: coderImg + ":" + coderVersion,
		Env: []string{
			fmt.Sprintf("CODER_ACCESS_URL=http://host.docker.internal:%d", p),
			"CODER_IN_MEMORY=true",
			"CODER_TELEMETRY_ENABLE=false",
			"TF_CLI_CONFIG_FILE=/tmp/integration.tfrc",
		},
		Labels: map[string]string{},
	}, &container.HostConfig{
		Binds: []string{
			tfrcPath + ":/tmp/integration.tfrc",
			srcPath + ":/src",
		},
	}, nil, nil, "")
	require.NoError(t, err, "create test deployment")

	t.Logf("created container %s\n", ctr.ID)
	t.Cleanup(func() {
		t.Logf("stopping container %s\n", ctr.ID)
		_ = cli.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{
			Force: true,
		})
	})

	err = cli.ContainerStart(ctx, ctr.ID, container.StartOptions{})
	require.NoError(t, err, "start container")
	t.Logf("started container %s\n", ctr.ID)

	// Perform first time setup
	var (
		testEmail    = "testing@coder.com"
		testPassword = "InsecurePassw0rd!"
		testUsername = "testing"
	)

	// Wait for container to come up
	execContainer(ctx, t, ctr.ID, `until curl -s --fail http://localhost:3000/healthz; do sleep 1; done`)
	// Perform first time setup
	execContainer(ctx, t, ctr.ID, fmt.Sprintf(`coder login http://localhost:3000 --first-user-email=%q --first-user-password=%q --first-user-trial=false --first-user-username=%q`, testEmail, testPassword, testUsername))
	return ctr.ID
}

// execContainer executes the given command in the given container and returns
// the output.
func execContainer(ctx context.Context, t *testing.T, containerID, command string) (string, int) {
	t.Helper()
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

func randomPort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen on random port")
	defer func() {
		_ = l.Close()
	}()
	tcpAddr, valid := l.Addr().(*net.TCPAddr)
	require.True(t, valid, "net.Listen did not return a *net.TCPAddr")
	return tcpAddr.Port
}
