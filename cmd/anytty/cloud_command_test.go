package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCloudStatusDisableEnableOffline(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	socketPath := filepath.Join(root, "anytty.sock")
	logPath := filepath.Join(root, "anytty.log")

	status := executeCloudStatusJSON(t, "--socket", socketPath, "--log-file", logPath, "cloud", "status", "--json")
	if status.State != "not_enrolled" || !status.Enabled || status.Enrolled || status.Running {
		t.Fatalf("initial cloud status = %#v", status)
	}
	if _, err := os.Stat(daemonRecordPath(socketPath)); !os.IsNotExist(err) {
		t.Fatalf("cloud status unexpectedly touched daemon runtime record: %v", err)
	}

	disabled := executeCloudStatusJSON(t, "--socket", socketPath, "--log-file", logPath, "cloud", "disable", "--json")
	if disabled.State != "disabled" || disabled.Enabled || disabled.Running {
		t.Fatalf("disabled cloud status = %#v", disabled)
	}
	if _, err := os.Stat(v3CloudDisabledPath()); err != nil {
		t.Fatalf("disabled marker was not written: %v", err)
	}

	enabled := executeCloudStatusJSON(t, "--socket", socketPath, "--log-file", logPath, "cloud", "enable", "--json")
	if enabled.State != "not_enrolled" || !enabled.Enabled || enabled.Enrolled || enabled.Running {
		t.Fatalf("enabled cloud status = %#v", enabled)
	}
	if _, err := os.Stat(v3CloudDisabledPath()); !os.IsNotExist(err) {
		t.Fatalf("disabled marker remains after enable: %v", err)
	}
}

func TestWriteCloudEdgeSelectionIncludesScore(t *testing.T) {
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	selection := &cloudv1.DaemonEdgeSelection{
		DaemonId: "daemon-test", SelectedEdgeId: "edge-test",
		Candidates: []*cloudv1.DaemonEdgeCandidate{{
			Locator: &cloudv1.EdgeLocator{EdgeId: "edge-test", Name: "CN2", Region: "CN2"},
			Score:   -123.4, Status: "可用",
		}},
	}
	if err := writeCloudEdgeSelection(command, selection); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "SCORE") || !strings.Contains(output.String(), "-123.4") {
		t.Fatalf("Edge selection output missing score:\n%s", output.String())
	}
}

func TestCloudEnrollDoesNotExposeControllerFlags(t *testing.T) {
	var socket, logFile, configPath string
	command := cloudEnrollCommand(&socket, &logFile, &configPath)
	for _, name := range []string{"controller", "controller-address", "controller-server-name"} {
		if command.Flags().Lookup(name) != nil {
			t.Fatalf("cloud enroll still exposes --%s", name)
		}
	}
}

func TestCloudEnrollmentProgressShowsRoutingWithoutPrivateCapacity(t *testing.T) {
	var output bytes.Buffer
	locator := &cloudv1.EdgeLocator{EdgeId: "edge-cn2", Name: "CN2", Region: "CN2", PublicEndpoint: "cn2.example.com:41102"}
	measurement := &cloudv1.DaemonEdgeMeasurement{EdgeId: locator.GetEdgeId(), Reachable: true, ConnectLatencyMs: 42, SampleCount: 3}
	candidate := &cloudv1.DaemonEdgeCandidate{Locator: locator, Online: true, Eligible: true, Measurement: measurement, Score: 1000, Status: "available"}
	writeCloudEnrollmentProgress(&output, clouddaemon.EnrollmentProgress{Stage: clouddaemon.EnrollmentProgressCandidates, Candidates: []*cloudv1.DaemonEdgeCandidate{candidate}})
	writeCloudEnrollmentProgress(&output, clouddaemon.EnrollmentProgress{Stage: clouddaemon.EnrollmentProgressMeasured, Candidates: []*cloudv1.DaemonEdgeCandidate{candidate}, Measurements: []*cloudv1.DaemonEdgeMeasurement{measurement}})
	writeCloudEnrollmentProgress(&output, clouddaemon.EnrollmentProgress{Stage: clouddaemon.EnrollmentProgressSelected, Selection: &cloudv1.DaemonEdgeSelection{SelectedEdgeId: locator.GetEdgeId(), Candidates: []*cloudv1.DaemonEdgeCandidate{candidate}}, SelectedEdge: locator})
	logged := output.String()
	for _, expected := range []string{"CN2", "cn2.example.com:41102", "latency=42 ms", "score=1000.0", "selected CN2"} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("enrollment progress %q does not contain %q", logged, expected)
		}
	}
	for _, private := range []string{"capacity", "agent_count", "load"} {
		if strings.Contains(strings.ToLower(logged), private) {
			t.Fatalf("enrollment progress exposed %q: %s", private, logged)
		}
	}
}

func TestCloudEntitlementRuntimeStatusIsActionable(t *testing.T) {
	state, detail := cloudEntitlementRuntimeStatus(&cloudv1.CloudEntitlementFailure{
		Code:  cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED,
		Limit: 2,
	})
	if state != "quota_limited" || !strings.Contains(detail, "limit 2") || !strings.Contains(detail, "cloud.anytty.com/app/subscription") || !strings.Contains(detail, "Direct and SSH") {
		t.Fatalf("daemon limit status = %q / %q", state, detail)
	}

	state, detail = cloudEntitlementRuntimeStatus(&cloudv1.CloudEntitlementFailure{Code: cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE})
	if state != "subscription_inactive" || !strings.Contains(detail, "renew") {
		t.Fatalf("subscription status = %q / %q", state, detail)
	}
}

func executeCloudStatusJSON(t *testing.T, args ...string) cloudStatusView {
	t.Helper()
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("anytty %v: %v\n%s", args, err, output.String())
	}
	var status cloudStatusView
	if err := json.Unmarshal(output.Bytes(), &status); err != nil {
		t.Fatalf("decode cloud status %q: %v", output.String(), err)
	}
	return status
}
