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
