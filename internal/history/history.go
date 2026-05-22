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
// digest, when non-empty, is stored as the first line of the commit message
// body: "digest: <value>".
func (m *Manager) Commit(decisions []byte, msgPrefix, digest string) error {
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
	var bodyParts []string
	if digest != "" {
		bodyParts = append(bodyParts, "digest: "+digest)
	}
	if len(decisions) > 0 {
		bodyParts = append(bodyParts, string(decisions))
	}
	if len(bodyParts) > 0 {
		msg += "\n\n" + strings.Join(bodyParts, "\n")
	}

	if err := runGit(m.repoPath, "commit", "--allow-empty", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// ── Log and Show ─────────────────────────────────────────────────────────────

// LogEntry describes a single commit in the history repository.
type LogEntry struct {
	Hash      string    // abbreviated git hash
	Timestamp time.Time // commit author timestamp
	Subject   string    // first line of commit message
	Decisions string    // one-line summary of decisions (key→value), empty for non-submit commits
	Digest    string    // content digest embedded in commit body, empty when absent
}

// Round holds the spec bytes and optional decisions bytes at a given ref.
type Round struct {
	Spec      []byte // contents of the .speckle file at that ref
	Decisions []byte // contents of the .decisions.json sidecar, may be nil
}

// Log returns all commits in the history repository, newest first.
// If no commits exist yet it returns an empty (non-nil) slice.
func (m *Manager) Log() ([]LogEntry, error) {
	// Use ASCII record separator (0x1e) between commits, and unit separator
	// (0x1f) between fields within a commit: hash\x1ftimestamp\x1fsubject\x1fbody\x1e
	const format = "%h\x1f%aI\x1f%s\x1f%b\x1e"
	cmd := exec.Command("git", "-C", m.repoPath, "log", "--format="+format)
	out, err := cmd.Output()
	if err != nil {
		// No commits yet: git log exits non-zero on empty repo.
		return []LogEntry{}, nil
	}
	raw := string(out)
	if strings.TrimSpace(raw) == "" {
		return []LogEntry{}, nil
	}

	var entries []LogEntry
	for _, rec := range strings.Split(raw, "\x1e") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\x1f", 4)
		if len(parts) < 3 {
			continue
		}
		hash := strings.TrimSpace(parts[0])
		tsStr := strings.TrimSpace(parts[1])
		subject := strings.TrimSpace(parts[2])
		body := ""
		if len(parts) == 4 {
			body = strings.TrimSpace(parts[3])
		}

		if hash == "" {
			continue
		}

		ts, _ := time.Parse(time.RFC3339, tsStr)

		decisions := ""
		digest := ""
		if body != "" {
			decisions = decisionsOneLiner(body)
			digest = digestFromBody(body)
		}

		entries = append(entries, LogEntry{
			Hash:      hash,
			Timestamp: ts,
			Subject:   subject,
			Decisions: decisions,
			Digest:    digest,
		})
	}
	return entries, nil
}

// digestFromBody extracts the "digest: <value>" line from the commit body,
// or returns "" if not present.
func digestFromBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "digest: ") {
			return strings.TrimPrefix(line, "digest: ")
		}
	}
	return ""
}

// decisionsOneLiner extracts key→value pairs from a decisions JSON body,
// returning a compact one-line summary. Returns "" if body isn't valid JSON.
func decisionsOneLiner(body string) string {
	// Find the first '{' to skip any preamble
	start := strings.Index(body, "{")
	if start < 0 {
		return ""
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body[start:]), &top); err != nil {
		return ""
	}
	decRaw, ok := top["decisions"]
	if !ok {
		return ""
	}
	var decisions map[string]json.RawMessage
	if err := json.Unmarshal(decRaw, &decisions); err != nil {
		return ""
	}
	var parts []string
	for k, v := range decisions {
		// Each value is typically {"selected":"x",...}
		var dec struct {
			Selected *string `json:"selected"`
		}
		if err := json.Unmarshal(v, &dec); err == nil && dec.Selected != nil {
			parts = append(parts, k+"="+*dec.Selected)
		} else {
			parts = append(parts, k+"=?")
		}
	}
	return strings.Join(parts, " ")
}

// Show returns the spec and optional decisions at a given git ref.
// Returns an error if the ref does not exist.
func (m *Manager) Show(ref string) (*Round, error) {
	base := filepath.Base(m.specPath)

	// Read spec at ref
	specContent, err := gitShow(m.repoPath, ref+":"+base)
	if err != nil {
		return nil, fmt.Errorf("show spec at %s: %w", ref, err)
	}

	// Try to read decisions sidecar
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	sidecarName := stem + ".decisions.json"
	var decisionsContent []byte
	if dc, err := gitShow(m.repoPath, ref+":"+sidecarName); err == nil {
		decisionsContent = dc
	}

	return &Round{
		Spec:      specContent,
		Decisions: decisionsContent,
	}, nil
}

// gitShow runs git show <object> in repoPath and returns the contents.
func gitShow(repoPath, object string) ([]byte, error) {
	cmd := exec.Command("git", "-C", repoPath, "show", object)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s: %w", object, err)
	}
	return out, nil
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
