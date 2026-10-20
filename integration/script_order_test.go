package integration

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Matches the sleep in coder-script-order/main.tf.
const cloneRepoSleep = 3 * time.Second

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
