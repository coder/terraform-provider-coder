package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These mirror the codersdk types of the same name, keeping only the fields
// the script ordering cases read. The package does not import codersdk, see
// the note on TestIntegration.
type workspaceAgentScriptDependency struct {
	PrerequisiteResourceAddress string `json:"prerequisite_resource_address"`
	Requirement                 string `json:"requirement"`
}

type workspaceAgentScript struct {
	DisplayName     string                           `json:"display_name"`
	LogSourceID     string                           `json:"log_source_id"`
	ResourceAddress string                           `json:"resource_address"`
	Dependencies    []workspaceAgentScriptDependency `json:"dependencies"`
}

type workspaceAgent struct {
	LifecycleState string                 `json:"lifecycle_state"`
	Scripts        []workspaceAgentScript `json:"scripts"`
}

type agentScriptTiming struct {
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at"`
	ExitCode    int32     `json:"exit_code"`
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
	DisplayName string    `json:"display_name"`
}

type workspaceBuildTimings struct {
	AgentScriptTimings []agentScriptTiming `json:"agent_script_timings"`
}

const (
	agentWaitTimeout  = 2 * time.Minute
	reportWaitTimeout = 30 * time.Second
	// Matches the sleep in coder-script-order/main.tf.
	cloneRepoSleep = 3 * time.Second
)

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

// stopAgent sends SIGTERM to the case's agent.
func stopAgent(ctx context.Context, t *testing.T, ctrID, name string) {
	t.Helper()
	// Never signal the process by name: the server in the same container is
	// the same binary.
	_, rc := execContainer(ctx, t, ctrID, fmt.Sprintf("kill -TERM $(cat /tmp/%s-agent.pid)", name))
	require.Equal(t, 0, rc, "stop agent for %s", name)
}

// agentLogTail returns the last lines of the agent's own log file.
func agentLogTail(ctx context.Context, t *testing.T, ctrID, name string) string {
	t.Helper()
	tail, _ := execContainer(ctx, t, ctrID, fmt.Sprintf("tail -n 50 /tmp/%s-agent.log", name))
	return tail
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
	t.Fatalf("agent %s did not reach %v in time: last state %q, last error %v\nagent log tail:\n%s",
		name, want, agent.LifecycleState, lastErr, agentLogTail(ctx, t, ctrID, name))
	return agent
}

// requireLifecycle fails with the agent log tail when the agent ended in a
// different state, since the log names the script that caused it.
func requireLifecycle(ctx context.Context, t *testing.T, ctrID, name string, agent workspaceAgent, want string) {
	t.Helper()
	if agent.LifecycleState != want {
		t.Fatalf("agent %s ended in %q, want %q\nagent log tail:\n%s",
			name, agent.LifecycleState, want, agentLogTail(ctx, t, ctrID, name))
	}
}

// waitForOrderLog polls the case's order log until it contains every line in
// want.
func waitForOrderLog(ctx context.Context, t *testing.T, ctrID, name string, want ...string) {
	t.Helper()
	ok := assert.Eventually(t, func() bool {
		// The condition runs on testify's goroutine, where a failed require
		// would stall the poll instead of ending the test.
		o, _, err := tryExecContainer(ctx, t, ctrID, fmt.Sprintf("cat /tmp/%s.log 2>/dev/null", name))
		if err != nil {
			return false
		}
		for _, w := range want {
			if !strings.Contains(o, w) {
				return false
			}
		}
		return true
	}, agentWaitTimeout, time.Second)
	if !ok {
		t.Fatalf("order log for %s never contained all of %q\nagent log tail:\n%s",
			name, want, agentLogTail(ctx, t, ctrID, name))
	}
}

// orderLogLines returns the lines the case's scripts appended to the order
// log, in the order they were written.
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

// waitForTimings polls the latest build's timings until every named script
// has one timing for the stage.
func waitForTimings(ctx context.Context, t *testing.T, ctrID, name, workspaceID, stage string, displayNames ...string) workspaceBuildTimings {
	t.Helper()
	var ws struct {
		LatestBuild struct {
			ID string `json:"id"`
		} `json:"latest_build"`
	}
	require.NoError(t, apiGet(ctx, t, ctrID, name, "/api/v2/workspaces/"+workspaceID, &ws))
	// The agent reports its lifecycle state before the goroutines that send
	// each timing have finished, so the first read after a state change can
	// be empty.
	var timings workspaceBuildTimings
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		timings = workspaceBuildTimings{}
		if !assert.NoError(c, apiGet(ctx, t, ctrID, name, "/api/v2/workspacebuilds/"+ws.LatestBuild.ID+"/timings", &timings)) {
			return
		}
		for _, dn := range displayNames {
			n := 0
			for _, tm := range timings.AgentScriptTimings {
				if tm.Stage == stage && tm.DisplayName == dn {
					n++
				}
			}
			assert.Equalf(c, 1, n, "expected one %s timing for %q, got %+v", stage, dn, timings.AgentScriptTimings)
		}
	}, reportWaitTimeout, time.Second)
	return timings
}

// timingFor returns the one timing with the given stage and display name.
func timingFor(t *testing.T, timings workspaceBuildTimings, stage, displayName string) agentScriptTiming {
	t.Helper()
	var found []agentScriptTiming
	for _, tm := range timings.AgentScriptTimings {
		if tm.Stage == stage && tm.DisplayName == displayName {
			found = append(found, tm)
		}
	}
	require.Lenf(t, found, 1, "expected one %s timing for %q, got %+v", stage, displayName, found)
	return found[0]
}

func requireStatus(t *testing.T, tm agentScriptTiming, want string) {
	t.Helper()
	require.Equalf(t, want, tm.Status, "%s: %+v", tm.DisplayName, tm)
}

func requireExitCode(t *testing.T, tm agentScriptTiming, want int32) {
	t.Helper()
	require.Equalf(t, want, tm.ExitCode, "%s: %+v", tm.DisplayName, tm)
}

// scriptFor fails the test if no script has displayName.
func scriptFor(t *testing.T, agent workspaceAgent, displayName string) workspaceAgentScript {
	t.Helper()
	for _, s := range agent.Scripts {
		if s.DisplayName == displayName {
			return s
		}
	}
	t.Fatalf("no script named %q in %+v", displayName, agent.Scripts)
	return workspaceAgentScript{}
}

// requireDependencies asserts a script's resource address and its stored
// dependencies. A field renamed on either side of the provider and Coder
// contract shows up here as a script with no dependencies.
func requireDependencies(t *testing.T, agent workspaceAgent, displayName, address string, want []workspaceAgentScriptDependency) {
	t.Helper()
	s := scriptFor(t, agent, displayName)
	require.Equalf(t, address, s.ResourceAddress, "%s resource_address", displayName)
	require.Equalf(t, want, s.Dependencies, "%s dependencies", displayName)
}

func verifyScriptOrder(ctx context.Context, t *testing.T, ctrID, name string, out map[string]string) {
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "ready", "start_error")

	// Checked before the order so a contract break is named as such rather
	// than as a script running too early.
	requireDependencies(t, agent, "Clone repo", "coder_script.clone_repo", nil)
	requireDependencies(t, agent, "Install tools", "coder_script.install_tools",
		[]workspaceAgentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "success"}})
	requireDependencies(t, agent, "Post setup", "coder_script.post_setup",
		[]workspaceAgentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "completion"}})
	requireDependencies(t, agent, "code-server", "coder_script.start_code_server", nil)

	requireLifecycle(ctx, t, ctrID, name, agent, "ready")

	lines := orderLogLines(ctx, t, ctrID, name)
	require.Len(t, lines, 8, "expected two lines per script: %q", lines)
	requireLineAfter(t, lines, "install_tools start", "clone_repo end")
	requireLineAfter(t, lines, "post_setup start", "clone_repo end")
	requireLineAfter(t, lines, "clone_repo end", "start_code_server start")

	timings := waitForTimings(ctx, t, ctrID, name, out["workspace.id"], "start", "Clone repo", "Install tools", "Post setup", "code-server")
	clone := timingFor(t, timings, "start", "Clone repo")
	install := timingFor(t, timings, "start", "Install tools")
	post := timingFor(t, timings, "start", "Post setup")
	code := timingFor(t, timings, "start", "code-server")
	for _, tm := range []agentScriptTiming{clone, install, post, code} {
		requireStatus(t, tm, "ok")
		requireExitCode(t, tm, 0)
	}
	require.Falsef(t, install.StartedAt.Before(clone.EndedAt), "Install tools started %s before Clone repo ended %s", install.StartedAt, clone.EndedAt)
	require.Falsef(t, post.StartedAt.Before(clone.EndedAt), "Post setup started %s before Clone repo ended %s", post.StartedAt, clone.EndedAt)
	require.Truef(t, code.StartedAt.Before(clone.EndedAt), "code-server started %s after Clone repo ended %s", code.StartedAt, clone.EndedAt)
	require.GreaterOrEqualf(t, clone.EndedAt.Sub(clone.StartedAt), cloneRepoSleep, "Clone repo duration: %+v", clone)
}

func verifyScriptOrderSkip(ctx context.Context, t *testing.T, ctrID, name string, out map[string]string) {
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "ready", "start_error")

	requireDependencies(t, agent, "Install tools", "coder_script.install_tools",
		[]workspaceAgentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "success"}})

	requireLifecycle(ctx, t, ctrID, name, agent, "start_error")

	lines := orderLogLines(ctx, t, ctrID, name)
	require.NotContains(t, lines, "install_tools start", "skipped script must not run: %q", lines)
	requireLineAfter(t, lines, "post_setup start", "clone_repo end")

	timings := waitForTimings(ctx, t, ctrID, name, out["workspace.id"], "start", "Clone repo", "Install tools", "Post setup", "code-server")
	clone := timingFor(t, timings, "start", "Clone repo")
	requireStatus(t, clone, "exit_failure")
	requireExitCode(t, clone, 1)
	install := timingFor(t, timings, "start", "Install tools")
	requireStatus(t, install, "skipped")
	require.Truef(t, install.StartedAt.Equal(install.EndedAt), "skipped timing must be zero-length: %+v", install)
	requireStatus(t, timingFor(t, timings, "start", "Post setup"), "ok")
	requireStatus(t, timingFor(t, timings, "start", "code-server"), "ok")
}

func verifyScriptOrderStop(ctx context.Context, t *testing.T, ctrID, name string, out map[string]string) {
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "ready")

	requireDependencies(t, agent, "Stop DB", "coder_script.stop_db",
		[]workspaceAgentScriptDependency{{PrerequisiteResourceAddress: "coder_script.save_state", Requirement: "success"}})

	stopAgent(ctx, t, ctrID, name)
	agent = waitForAgent(ctx, t, ctrID, name, agentID, "off", "shutdown_error", "shutdown_timeout")
	requireLifecycle(ctx, t, ctrID, name, agent, "off")

	lines := orderLogLines(ctx, t, ctrID, name)
	requireLineAfter(t, lines, "stop_db start", "save_state end")

	timings := waitForTimings(ctx, t, ctrID, name, out["workspace.id"], "stop", "Save state", "Stop DB")
	save := timingFor(t, timings, "stop", "Save state")
	stop := timingFor(t, timings, "stop", "Stop DB")
	requireStatus(t, save, "ok")
	requireStatus(t, stop, "ok")
	require.Falsef(t, stop.StartedAt.Before(save.EndedAt), "Stop DB started %s before Save state ended %s", stop.StartedAt, save.EndedAt)
}

func verifyScriptOrderCancel(ctx context.Context, t *testing.T, ctrID, name string, out map[string]string) {
	agentID := agentIDFor(ctx, t, ctrID, name, out["workspace.id"])
	startAgent(ctx, t, ctrID, name)
	// code-server starts in parallel with clone_repo and must have exited
	// before the signal, otherwise it is canceled too.
	waitForOrderLog(ctx, t, ctrID, name, "clone_repo start", "start_code_server end")
	stopAgent(ctx, t, ctrID, name)
	agent := waitForAgent(ctx, t, ctrID, name, agentID, "off", "shutdown_error", "shutdown_timeout")

	requireDependencies(t, agent, "Install tools", "coder_script.install_tools",
		[]workspaceAgentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "success"}})
	requireDependencies(t, agent, "Post setup", "coder_script.post_setup",
		[]workspaceAgentScriptDependency{{PrerequisiteResourceAddress: "coder_script.clone_repo", Requirement: "completion"}})

	requireLifecycle(ctx, t, ctrID, name, agent, "off")

	lines := orderLogLines(ctx, t, ctrID, name)
	require.NotContains(t, lines, "clone_repo end")
	require.NotContains(t, lines, "install_tools start")
	require.NotContains(t, lines, "post_setup start")

	timings := waitForTimings(ctx, t, ctrID, name, out["workspace.id"], "start", "Clone repo", "Install tools", "Post setup", "code-server")
	clone := timingFor(t, timings, "start", "Clone repo")
	requireStatus(t, clone, "canceled")
	require.Truef(t, clone.EndedAt.After(clone.StartedAt), "canceled running script must keep its real interval: %+v", clone)
	for _, n := range []string{"Install tools", "Post setup"} {
		tm := timingFor(t, timings, "start", n)
		requireStatus(t, tm, "canceled")
		require.Truef(t, tm.StartedAt.Equal(tm.EndedAt), "%s timing must be zero-length: %+v", n, tm)
	}
	requireStatus(t, timingFor(t, timings, "start", "code-server"), "ok")
}
