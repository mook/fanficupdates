package calibre

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/mook/fanficupdates/model"
)

type Calibre struct {
	library  string // Path to the Calibre library
	settings string // Path to the settings directory
	override string // Overridden output, for testing
}

func (c *Calibre) WithLibrary(libraryPath string) *Calibre {
	c.library = libraryPath
	return c
}

func (c *Calibre) WithSettings(settingsPath string) *Calibre {
	c.settings = settingsPath
	return c
}

func (c *Calibre) WithOverride(override string) *Calibre {
	c.override = override
	return c
}

// Run calibredb with the given arguments, returning stdout.
func (c *Calibre) Run(ctx context.Context, args ...string) (string, error) {
	if c.override != "" {
		return c.override, nil
	}

	buf := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "calibredb")
	if c.library != "" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--library-path=%s", c.library))
	}
	cmd.Args = append(cmd.Args, args...)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	if c.settings != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CALIBRE_CONFIG_DIRECTORY=%s", c.settings))
	}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Calibre) GetBooks(ctx context.Context) ([]model.CalibreBook, error) {
	data, err := c.Run(ctx, "list", "--for-machine", "--fields=all")
	if err != nil {
		return nil, err
	}
	var result []model.CalibreBook
	if err = json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}
