package nab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNotAvailable = errors.New("nab not available")

var (
	lookPath       = exec.LookPath
	commandContext = exec.CommandContext
)

type Client struct {
	path string
}

type fetchResponse struct {
	Status   int    `json:"status"`
	Markdown string `json:"markdown"`
}

func LookupPath() (string, error) {
	path, err := lookPath("nab")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	return path, nil
}

func New() (*Client, error) {
	path, err := LookupPath()
	if err != nil {
		return nil, err
	}
	return &Client{path: path}, nil
}

func (c *Client) Available() bool {
	return c != nil && c.path != ""
}

// FetchHTML returns the HTML payload from `nab fetch --format json --raw-html`.
// nab emits that payload in the "markdown" field even when the content is raw HTML.
func (c *Client) FetchHTML(ctx context.Context, rawURL, browser string) ([]byte, error) {
	if !c.Available() {
		return nil, ErrNotAvailable
	}
	if browser == "" {
		browser = "auto"
	}

	cmd := commandContext(
		ctx,
		c.path,
		"fetch",
		rawURL,
		"--format",
		"json",
		"--raw-html",
		"--cookies",
		browser,
		"--no-save",
	)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("nab fetch: %s", stderr)
			}
		}
		return nil, fmt.Errorf("nab fetch: %w", err)
	}

	var resp fetchResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &resp); err != nil {
		return nil, fmt.Errorf("nab fetch json: %w", err)
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("nab fetch status %d", resp.Status)
	}
	return []byte(resp.Markdown), nil
}
