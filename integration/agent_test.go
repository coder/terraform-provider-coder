package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// These mirror the fields of the Coder API that the script ordering cases
// read. The package does not import codersdk, see the note on TestIntegration.
type agentScriptDependency struct {
	PrerequisiteResourceAddress string `json:"prerequisite_resource_address"`
	Requirement                 string `json:"requirement"`
}

type agentScript struct {
	DisplayName     string                  `json:"display_name"`
	LogSourceID     string                  `json:"log_source_id"`
	ResourceAddress string                  `json:"resource_address"`
	Dependencies    []agentScriptDependency `json:"dependencies"`
}

type workspaceAgent struct {
	LifecycleState string        `json:"lifecycle_state"`
	Scripts        []agentScript `json:"scripts"`
}

type scriptTiming struct {
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at"`
	ExitCode    int32     `json:"exit_code"`
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
	DisplayName string    `json:"display_name"`
}

type buildTimings struct {
	AgentScriptTimings []scriptTiming `json:"agent_script_timings"`
}

type agentLog struct {
	Output   string `json:"output"`
	SourceID string `json:"source_id"`
}

const agentWaitTimeout = 2 * time.Minute

// startAgent runs coder agent in the background inside the Coder container
// for the workspace the named case created.
func startAgent(ctx context.Context, t *testing.T, ctrID, name string) {
	t.Helper()
	// The agent reads its token from the file the template wrote, so the
	// token never appears in the command the harness logs. The subshell
	// execs the agent so the PID file holds the agent's own PID.
	cmd := fmt.Sprintf(
		`mkdir -p /tmp/%[1]s-agent && `+
			`( exec env CODER_AGENT_URL=http://localhost:3000 `+
			`CODER_AGENT_TOKEN_FILE=/tmp/%[1]s.agent-token `+
			`CODER_AGENT_LOG_DIR=/tmp/%[1]s-agent `+
			`CODER_AGENT_SCRIPT_DATA_DIR=/tmp/%[1]s-agent `+
			`coder agent >/tmp/%[1]s-agent.log 2>&1 ) & `+
			`echo $! >/tmp/%[1]s-agent.pid`,
		name)
	_, rc := execContainer(ctx, t, ctrID, cmd)
	require.Equal(t, 0, rc, "start agent for %s", name)
}

// stopAgent sends SIGTERM to the case's agent so it runs its stop scripts.
func stopAgent(ctx context.Context, t *testing.T, ctrID, name string) {
	t.Helper()
	// Never signal the process by name: the server in the same container is
	// the same binary.
	_, rc := execContainer(ctx, t, ctrID, fmt.Sprintf("kill -TERM $(cat /tmp/%s-agent.pid)", name))
	require.Equal(t, 0, rc, "stop agent for %s", name)
}

// apiGet performs a GET against the Coder in the container as the workspace
// owner and decodes the JSON body into out.
func apiGet(ctx context.Context, t *testing.T, ctrID, name, path string, out any) error {
	t.Helper()
	// The session token is read by the container's shell at the moment of
	// the call, so it never appears in the command the harness logs.
	cmd := fmt.Sprintf(
		`curl -s -w '\n%%{http_code}' -H "Coder-Session-Token: $(cat /tmp/%s.session-token)" http://localhost:3000%s`,
		name, path)
	body, rc := execContainer(ctx, t, ctrID, cmd)
	if rc != 0 {
		return fmt.Errorf("GET %s: exit %d: %s", path, rc, body)
	}
	i := strings.LastIndex(body, "\n")
	if i < 0 {
		return fmt.Errorf("GET %s: no status line in %q", path, body)
	}
	status, body := body[i+1:], body[:i]
	if status != "200" {
		return fmt.Errorf("GET %s: HTTP %s: %s", path, status, body)
	}
	if err := json.Unmarshal([]byte(body), out); err != nil {
		return fmt.Errorf("decode GET %s: %w: %s", path, err, body)
	}
	return nil
}

// agentIDFor returns the ID coderd assigned to the workspace's one agent.
func agentIDFor(ctx context.Context, t *testing.T, ctrID, name, workspaceID string) string {
	t.Helper()
	// coderd generates its own agent ID on insert, so the ID in the
	// Terraform state is not the one the API knows.
	var ws struct {
		LatestBuild struct {
			Resources []struct {
				Agents []struct {
					ID string `json:"id"`
				} `json:"agents"`
			} `json:"resources"`
		} `json:"latest_build"`
	}
	require.NoError(t, apiGet(ctx, t, ctrID, name, "/api/v2/workspaces/"+workspaceID, &ws))
	var ids []string
	for _, r := range ws.LatestBuild.Resources {
		for _, a := range r.Agents {
			ids = append(ids, a.ID)
		}
	}
	require.Lenf(t, ids, 1, "expected one agent on workspace %s, got %v", workspaceID, ids)
	return ids[0]
}

// waitForAgent polls the agent until its lifecycle state is one of want and
// returns the last response.
func waitForAgent(ctx context.Context, t *testing.T, ctrID, name, agentID string, want ...string) workspaceAgent {
	t.Helper()
	deadline := time.Now().Add(agentWaitTimeout)
	var agent workspaceAgent
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = apiGet(ctx, t, ctrID, name, "/api/v2/workspaceagents/"+agentID, &agent)
		if lastErr == nil && slices.Contains(want, agent.LifecycleState) {
			return agent
		}
		time.Sleep(time.Second)
	}
	tail, _ := execContainer(ctx, t, ctrID, fmt.Sprintf("tail -n 50 /tmp/%s-agent.log", name))
	t.Fatalf("agent %s did not reach %v in time: last state %q, last error %v\nagent log tail:\n%s",
		name, want, agent.LifecycleState, lastErr, tail)
	return agent
}

// orderLogLines returns the lines the case's scripts appended to the durable
// file, in the order they were written.
func orderLogLines(ctx context.Context, t *testing.T, ctrID, name string) []string {
	t.Helper()
	out, rc := execContainer(ctx, t, ctrID, fmt.Sprintf("cat /tmp/%s.log", name))
	require.Equal(t, 0, rc, "read order log for %s", name)
	return strings.Split(strings.TrimSpace(out), "\n")
}

// requireLineAfter asserts that later appears exactly once and after earlier.
func requireLineAfter(t *testing.T, lines []string, later, earlier string) {
	t.Helper()
	// Only relative order is checked. Scripts that start together may write
	// in either order.
	laterIdx := slices.Index(lines, later)
	earlierIdx := slices.Index(lines, earlier)
	require.NotEqualf(t, -1, laterIdx, "line %q missing from %q", later, lines)
	require.NotEqualf(t, -1, earlierIdx, "line %q missing from %q", earlier, lines)
	require.Equalf(t, 1, countOf(lines, later), "line %q appears more than once in %q", later, lines)
	require.Greaterf(t, laterIdx, earlierIdx, "%q (line %d) should come after %q (line %d): %q", later, laterIdx, earlier, earlierIdx, lines)
}

func countOf(lines []string, want string) int {
	n := 0
	for _, l := range lines {
		if l == want {
			n++
		}
	}
	return n
}

// latestBuildTimings fetches the timings of the workspace's latest build.
func latestBuildTimings(ctx context.Context, t *testing.T, ctrID, name, workspaceID string) buildTimings {
	t.Helper()
	var ws struct {
		LatestBuild struct {
			ID string `json:"id"`
		} `json:"latest_build"`
	}
	require.NoError(t, apiGet(ctx, t, ctrID, name, "/api/v2/workspaces/"+workspaceID, &ws))
	var timings buildTimings
	require.NoError(t, apiGet(ctx, t, ctrID, name, "/api/v2/workspacebuilds/"+ws.LatestBuild.ID+"/timings", &timings))
	return timings
}

// timingFor returns the one timing with the given stage and display name.
func timingFor(t *testing.T, timings buildTimings, stage, displayName string) scriptTiming {
	t.Helper()
	var found []scriptTiming
	for _, tm := range timings.AgentScriptTimings {
		if tm.Stage == stage && tm.DisplayName == displayName {
			found = append(found, tm)
		}
	}
	require.Lenf(t, found, 1, "expected one %s timing for %q, got %+v", stage, displayName, found)
	return found[0]
}

// scriptFor returns the agent's script with the given display name.
func scriptFor(t *testing.T, agent workspaceAgent, displayName string) agentScript {
	t.Helper()
	for _, s := range agent.Scripts {
		if s.DisplayName == displayName {
			return s
		}
	}
	t.Fatalf("no script named %q in %+v", displayName, agent.Scripts)
	return agentScript{}
}

// requireDependencies asserts a script's resource address and its stored
// dependencies. A field renamed on either side of the provider and Coder
// contract shows up here as a script with no dependencies.
func requireDependencies(t *testing.T, agent workspaceAgent, displayName, address string, want []agentScriptDependency) {
	t.Helper()
	s := scriptFor(t, agent, displayName)
	require.Equalf(t, address, s.ResourceAddress, "%s resource_address", displayName)
	require.Equalf(t, want, s.Dependencies, "%s dependencies", displayName)
}

// agentLogLines returns the agent log lines for one script's log source.
func agentLogLines(ctx context.Context, t *testing.T, ctrID, name, agentID, sourceID string) []string {
	t.Helper()
	var logs []agentLog
	require.NoError(t, apiGet(ctx, t, ctrID, name, "/api/v2/workspaceagents/"+agentID+"/logs", &logs))
	var lines []string
	for _, l := range logs {
		if l.SourceID == sourceID {
			lines = append(lines, l.Output)
		}
	}
	return lines
}

func verifyScriptOrder(ctx context.Context, t *testing.T, ctrID string, out map[string]string) {
	const name = "coder-script-order"
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "ready", "start_error")
	require.Equal(t, "ready", agent.LifecycleState)

	lines := orderLogLines(ctx, t, ctrID, name)
	require.Len(t, lines, 8, "expected two lines per script: %q", lines)
	requireLineAfter(t, lines, "install_tools start", "clone_repo end")
	requireLineAfter(t, lines, "post_setup start", "clone_repo end")
	requireLineAfter(t, lines, "clone_repo end", "start_code_server start")

	timings := latestBuildTimings(ctx, t, ctrID, name, out["workspace.id"])
	clone := timingFor(t, timings, "start", "Clone repo")
	install := timingFor(t, timings, "start", "Install tools")
	post := timingFor(t, timings, "start", "Post setup")
	code := timingFor(t, timings, "start", "code-server")
	for _, tm := range []scriptTiming{clone, install, post, code} {
		require.Equalf(t, "ok", tm.Status, "%s: %+v", tm.DisplayName, tm)
		require.Equalf(t, int32(0), tm.ExitCode, "%s: %+v", tm.DisplayName, tm)
	}
	require.Falsef(t, install.StartedAt.Before(clone.EndedAt), "Install tools started %s before Clone repo ended %s", install.StartedAt, clone.EndedAt)
	require.Falsef(t, post.StartedAt.Before(clone.EndedAt), "Post setup started %s before Clone repo ended %s", post.StartedAt, clone.EndedAt)
	require.Truef(t, code.StartedAt.Before(clone.EndedAt), "code-server started %s after Clone repo ended %s", code.StartedAt, clone.EndedAt)
	require.GreaterOrEqual(t, clone.EndedAt.Sub(clone.StartedAt), 3*time.Second)

	requireDependencies(t, agent, "Clone repo", "coder_script.clone_repo", nil)
	requireDependencies(t, agent, "Install tools", "coder_script.install_tools",
		[]agentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "success"}})
	requireDependencies(t, agent, "Post setup", "coder_script.post_setup",
		[]agentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "completion"}})
	requireDependencies(t, agent, "code-server", "coder_script.start_code_server", nil)
}

func verifyScriptOrderSkip(ctx context.Context, t *testing.T, ctrID string, out map[string]string) {
	const name = "coder-script-order-skip"
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "ready", "start_error")
	require.Equal(t, "start_error", agent.LifecycleState)

	lines := orderLogLines(ctx, t, ctrID, name)
	require.NotContains(t, lines, "install_tools start", "skipped script must not run: %q", lines)
	requireLineAfter(t, lines, "post_setup start", "clone_repo end")

	timings := latestBuildTimings(ctx, t, ctrID, name, out["workspace.id"])
	clone := timingFor(t, timings, "start", "Clone repo")
	require.Equal(t, "exit_failure", clone.Status)
	require.Equal(t, int32(1), clone.ExitCode)
	install := timingFor(t, timings, "start", "Install tools")
	require.Equal(t, "skipped", install.Status)
	require.Truef(t, install.StartedAt.Equal(install.EndedAt), "skipped timing must be zero-length: %+v", install)
	require.Equal(t, "ok", timingFor(t, timings, "start", "Post setup").Status)
	require.Equal(t, "ok", timingFor(t, timings, "start", "code-server").Status)

	installLines := agentLogLines(ctx, t, ctrID, name, agentID, scriptFor(t, agent, "Install tools").LogSourceID)
	require.Contains(t, installLines, `Skipping script: dependency "Clone repo" failed.`)

	requireDependencies(t, agent, "Install tools", "coder_script.install_tools",
		[]agentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "success"}})
}

func verifyScriptOrderStop(ctx context.Context, t *testing.T, ctrID string, out map[string]string) {
	const name = "coder-script-order-stop"
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	waitForAgent(ctx, t, ctrID, name, agentID, "ready")
	stopAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "off", "shutdown_error", "shutdown_timeout")
	require.Equal(t, "off", agent.LifecycleState)

	lines := orderLogLines(ctx, t, ctrID, name)
	requireLineAfter(t, lines, "stop_db start", "save_state end")

	timings := latestBuildTimings(ctx, t, ctrID, name, out["workspace.id"])
	save := timingFor(t, timings, "stop", "Save state")
	stop := timingFor(t, timings, "stop", "Stop DB")
	require.Equal(t, "ok", save.Status)
	require.Equal(t, "ok", stop.Status)
	require.Falsef(t, stop.StartedAt.Before(save.EndedAt), "Stop DB started %s before Save state ended %s", stop.StartedAt, save.EndedAt)

	requireDependencies(t, agent, "Stop DB", "coder_script.stop_db",
		[]agentScriptDependency{{PrerequisiteResourceAddress: "coder_script.save_state", Requirement: "success"}})
}

func verifyScriptOrderCancel(ctx context.Context, t *testing.T, ctrID string, out map[string]string) {
	const name = "coder-script-order-cancel"
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	require.Eventually(t, func() bool {
		o, rc := execContainer(ctx, t, ctrID, fmt.Sprintf("cat /tmp/%s.log 2>/dev/null", name))
		return rc == 0 && strings.Contains(o, "clone_repo start")
	}, agentWaitTimeout, time.Second, "clone_repo never started")
	stopAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "off", "shutdown_error", "shutdown_timeout")

	lines := orderLogLines(ctx, t, ctrID, name)
	require.NotContains(t, lines, "clone_repo end")
	require.NotContains(t, lines, "install_tools start")
	require.NotContains(t, lines, "post_setup start")

	timings := latestBuildTimings(ctx, t, ctrID, name, out["workspace.id"])
	clone := timingFor(t, timings, "start", "Clone repo")
	require.Equal(t, "canceled", clone.Status)
	require.Truef(t, clone.EndedAt.After(clone.StartedAt), "canceled running script must keep its real interval: %+v", clone)
	for _, n := range []string{"Install tools", "Post setup"} {
		tm := timingFor(t, timings, "start", n)
		require.Equalf(t, "canceled", tm.Status, "%s: %+v", n, tm)
		require.Truef(t, tm.StartedAt.Equal(tm.EndedAt), "%s timing must be zero-length: %+v", n, tm)
	}
	require.Equal(t, "ok", timingFor(t, timings, "start", "code-server").Status)

	installLines := agentLogLines(ctx, t, ctrID, name, agentID, scriptFor(t, agent, "Install tools").LogSourceID)
	require.Contains(t, installLines, "Canceled before starting.")
	cloneLines := agentLogLines(ctx, t, ctrID, name, agentID, scriptFor(t, agent, "Clone repo").LogSourceID)
	require.Contains(t, cloneLines, "Canceled while running.")
}
