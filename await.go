package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func runAwait(args []string) error {
	fs := flag.NewFlagSet("await", flag.ContinueOnError)
	url := fs.String("url", "", "server URL (default: read from <file>.lock)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: speckle await [--url=URL] <file.speckle>")
	}
	path := fs.Arg(0)

	base := *url
	if base == "" {
		u, err := readLockURL(path)
		if err != nil {
			return fmt.Errorf("locate server: %w (start `speckle serve %s` first, or pass --url)", err, path)
		}
		base = u
	}

	client := &http.Client{Timeout: 0}
	req, err := http.NewRequest(http.MethodGet, base+"/await", nil)
	if err != nil {
		return err
	}
	// Set a long Connection: keep-alive, no client-side timeout — submission
	// may take an arbitrary amount of human time.
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, b)
	}
	var sub Submission
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return fmt.Errorf("decode submission: %w", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(sub)
}

func readLockURL(specPath string) (string, error) {
	b, err := os.ReadFile(specPath + ".lock")
	if err != nil {
		return "", err
	}
	var l struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(b, &l); err != nil {
		return "", err
	}
	if l.URL == "" {
		return "", errors.New("lockfile missing url")
	}
	return l.URL, nil
}

