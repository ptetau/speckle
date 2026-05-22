package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

const specWithDecidedAndOpen = `version: 1
title: UI Test Plan
dimensions:
  - id: cli
    label: CLI
    color: "#2c6fbb"
sections:
  - id: decided-section
    heading: Already Decided
    dimension: cli
    decisions:
      - id: decided-dec
        prompt: Already chosen?
        options:
          - id: opt-yes
            label: Yes
          - id: opt-no
            label: No
        selected: opt-yes
  - id: open-section
    heading: Open Decision
    dimension: cli
    decisions:
      - id: open-dec
        prompt: Still pending?
        options:
          - id: opt-a
            label: Option A
          - id: opt-b
            label: Option B
        selected: null
inbox:
  cli: "some idea for the inbox"
`

func TestSpecklerUIHasSidebarNav(t *testing.T) {
	url, _, stop := startServer(t, specWithDecidedAndOpen)
	defer stop()

	resp, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Page title should include "Speckle UI"
	if !strings.Contains(html, "Speckle UI") {
		t.Errorf("expected 'Speckle UI' in page title/header, got excerpt:\n%s", excerpt(html))
	}

	// Sidebar nav zones should be present
	if !strings.Contains(html, "INBOX") {
		t.Errorf("expected INBOX nav zone, got excerpt:\n%s", excerpt(html))
	}
	if !strings.Contains(html, "SPECS") {
		t.Errorf("expected SPECS nav zone, got excerpt:\n%s", excerpt(html))
	}
	if !strings.Contains(html, "DECISIONS") {
		t.Errorf("expected DECISIONS nav zone, got excerpt:\n%s", excerpt(html))
	}

	// Submit button should be in the header bar
	if !strings.Contains(html, "submit") {
		t.Errorf("expected submit button, got excerpt:\n%s", excerpt(html))
	}
}

func TestSpecklerUILockOnDecidedSections(t *testing.T) {
	url, _, stop := startServer(t, specWithDecidedAndOpen)
	defer stop()

	resp, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Decided sections should have lock indicator
	if !strings.Contains(html, "lock") && !strings.Contains(html, "decided") && !strings.Contains(html, "🔒") {
		t.Errorf("expected lock/decided indicator for decided sections, got excerpt:\n%s", excerpt(html))
	}
}

func TestSpecklerUIInboxSection(t *testing.T) {
	url, _, stop := startServer(t, specWithDecidedAndOpen)
	defer stop()

	resp, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Inbox content should appear when inbox has entries
	if !strings.Contains(html, "some idea for the inbox") {
		t.Errorf("expected inbox content in HTML, got excerpt:\n%s", excerpt(html))
	}
}
