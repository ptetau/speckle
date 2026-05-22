package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const specForExpand = `version: 1
title: Expand test
sections:
  - id: auth-sec
    heading: Auth section
    decisions:
      - id: auth-method
        prompt: Which auth approach?
        options:
          - id: jwt
            label: JWT
          - id: oauth
            label: OAuth
        selected: null
`

func TestExpandOutputsValidYAML(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specForExpand), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "expand", specPath, "auth-method")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle expand: %v\n%s", err, out)
	}

	stdout := string(out)

	// Should contain the decision ID
	if !strings.Contains(stdout, "auth-method") {
		t.Errorf("expand output should contain decision ID 'auth-method', got:\n%s", excerpt(stdout))
	}

	// Should be parseable as YAML (ignoring comment lines)
	var m interface{}
	// Strip comment lines for YAML parsing
	lines := strings.Split(stdout, "\n")
	var yamlLines []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			yamlLines = append(yamlLines, line)
		}
	}
	yamlStr := strings.Join(yamlLines, "\n")
	if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
		t.Errorf("expand output is not valid YAML: %v\noutput:\n%s", err, stdout)
	}
}

func TestExpandWithMode(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specForExpand), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "expand", "--mode=experimental", specPath, "auth-method")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle expand --mode=experimental: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "experimental") {
		t.Errorf("expected mode 'experimental' in output, got:\n%s", excerpt(string(out)))
	}
}

func TestExpandUnknownDecisionErrors(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specForExpand), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "expand", specPath, "nonexistent-decision")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for unknown decision, got none\n%s", out)
	}
}

func TestExpandUsageError(t *testing.T) {
	cmd := exec.Command(testBinary, "expand")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected usage error, got none\n%s", out)
	}
}

func TestServerExpandEndpoint(t *testing.T) {
	url, _, stop := startServer(t, specForExpand)
	defer stop()

	resp, err := http.Post(url+"/expand", "application/json",
		strings.NewReader(`{"decision_id":"auth-method","mode":"hybrid"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /expand: status %d\n%s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parse /expand JSON: %v\nbody: %s", err, body)
	}
	if result["overlay"] == "" {
		t.Errorf("expected non-empty overlay in response, got: %v", result)
	}
	if !strings.Contains(result["overlay"], "auth-method") {
		t.Errorf("overlay should contain decision ID 'auth-method', got:\n%s", result["overlay"])
	}
}
