// Package history manages git-backed history for .speckle spec files.
// Each spec gets a dedicated .speckle-repo git repository. A manifest
// file (.speckle-manifest.json) records the spec→repo mapping so that
// speckle can locate the correct history repo from any working directory.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	metaFileName     = ".speckle-meta.json"
	manifestFileName = ".speckle-manifest.json"
	defaultRepoName  = ".speckle-repo"
)

// Manager handles git history commits for one spec file.
type Manager struct {
	specPath     string // absolute path to the .speckle file
	repoPath     string // absolute path to the .speckle-repo git repository
	manifestPath string // absolute path to .speckle-manifest.json
}

// RepoPath returns the absolute path of the speckle git repository.
func (m *Manager) RepoPath() string { return m.repoPath }

// Open finds or initialises the history repository for specPath.
//
// Discovery order:
//  1. Walk up from the spec's directory to find a .speckle-manifest.json.
//     If an entry for this spec exists, use its configured history path.
//  2. Fall back to a .speckle-repo directory adjacent to the spec file.
//
// If no repository exists at the resolved path, Open initialises a fresh
// git repo, writes .speckle-meta.json (the detection marker), and updates
// (or creates) the manifest.
func Open(specPath string) (*Manager, error) {
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return nil, fmt.Errorf("history.Open: %w", err)
	}
	specDir := filepath.Dir(abs)

	// Prefer project git root for the manifest; fall back to spec directory.
	manifestDir := projectRoot(specDir)
	manifestPath := filepath.Join(manifestDir, manifestFileName)

	// Try to resolve history path from an existing manifest entry.
	repoPath := repoFromManifest(manifestPath, manifestDir, abs)
	if repoPath == "" {
		repoPath = filepath.Join(specDir, defaultRepoName)
	}

	if err := ensureRepo(repoPath); err != nil {
		return nil, err
	}
	if err := ensureManifestEntry(manifestPath, manifestDir, abs, repoPath); err != nil {
		return nil, err
	}

	return &Manager{
		specPath:     abs,
		repoPath:     repoPath,
		manifestPath: manifestPath,
	}, nil
}

// Commit copies the current spec file (and optional decisions bytes) into
// the history repository and creates a git commit.
//
// decisions may be nil; when provided they are written as a sidecar file
// named <spec-stem>.decisions.json and embedded in the commit message body.
// msgPrefix is used as the first-line suffix: "speckle: <msgPrefix>".
func (m *Manager) Commit(decisions []byte, msgPrefix string) error {
	base := filepath.Base(m.specPath)

	// Copy spec into repo.
	data, err := os.ReadFile(m.specPath)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	if err := os.WriteFile(filepath.Join(m.repoPath, base), data, 0o644); err != nil {
		return fmt.Errorf("copy spec to repo: %w", err)
	}

	// Write decisions sidecar when provided.
	if len(decisions) > 0 {
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		sidecar := filepath.Join(m.repoPath, stem+".decisions.json")
		if err := os.WriteFile(sidecar, decisions, 0o644); err != nil {
			return fmt.Errorf("write decisions sidecar: %w", err)
		}
	}

	if err := runGit(m.repoPath, "add", "."); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	msg := "speckle: " + msgPrefix
	if len(decisions) > 0 {
		msg += "\n\n" + string(decisions)
	}

	if err := runGit(m.repoPath, "commit", "--allow-empty", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────

type metaFile struct {
	Version int    `json:"version"`
	Created string `json:"created"`
}

type manifestFile struct {
	Specs []manifestEntry `json:"specs"`
}

type manifestEntry struct {
	File    string `json:"file"`    // relative path (slash-separated) from manifest dir to spec
	History string `json:"history"` // relative path (slash-separated) from manifest dir to repo
}

// projectRoot returns the git working-tree root containing dir, or dir
// itself when dir is not inside any git repository.
func projectRoot(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return dir
	}
	return strings.TrimSpace(string(out))
}

// repoFromManifest reads manifestPath and returns the absolute history repo
// path for specPath, or "" if no entry is found.
func repoFromManifest(manifestPath, manifestDir, specPath string) string {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ""
	}
	var m manifestFile
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	for _, e := range m.Specs {
		entrySpec := filepath.Join(manifestDir, filepath.FromSlash(e.File))
		if filepath.Clean(entrySpec) == filepath.Clean(specPath) {
			return filepath.Join(manifestDir, filepath.FromSlash(e.History))
		}
	}
	return ""
}

// ensureRepo creates and initialises the history repository at repoPath if
// it does not already contain a valid .speckle-meta.json marker.
func ensureRepo(repoPath string) error {
	if _, err := os.Stat(filepath.Join(repoPath, metaFileName)); err == nil {
		return nil // already initialised
	}
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		return fmt.Errorf("create repo dir: %w", err)
	}
	if err := runGit(repoPath, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	// Disable line-ending conversion to keep spec files byte-identical.
	_ = runGit(repoPath, "config", "core.autocrlf", "false")

	meta := metaFile{Version: 1, Created: time.Now().UTC().Format(time.RFC3339)}
	data, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(repoPath, metaFileName), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}
	return nil
}

// ensureManifestEntry adds a spec→repo entry to the manifest if none exists.
func ensureManifestEntry(manifestPath, manifestDir, specPath, repoPath string) error {
	var m manifestFile
	if data, err := os.ReadFile(manifestPath); err == nil {
		_ = json.Unmarshal(data, &m)
	}

	relSpec, _ := filepath.Rel(manifestDir, specPath)
	relRepo, _ := filepath.Rel(manifestDir, repoPath)
	relSpecSlash := filepath.ToSlash(relSpec)
	relRepoSlash := filepath.ToSlash(relRepo)

	for _, e := range m.Specs {
		if e.File == relSpecSlash {
			return nil // already present
		}
	}
	m.Specs = append(m.Specs, manifestEntry{File: relSpecSlash, History: relRepoSlash})

	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// runGit runs a git subcommand in dir with a fixed author identity so that
// commits succeed even when the global git config has no user set.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=speckle",
		"GIT_AUTHOR_EMAIL=speckle@localhost",
		"GIT_COMMITTER_NAME=speckle",
		"GIT_COMMITTER_EMAIL=speckle@localhost",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
