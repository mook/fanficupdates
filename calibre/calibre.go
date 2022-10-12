package calibre

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
)

type Calibre struct {
	library  string // Path to the Calibre library
	settings string // Path to the settings directory
	override string // Overridden output, for testing
}

type decodingBook struct {
	*model.CalibreBook
	Authors any // Override author field; see GetBooks() for details.
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

// Run the given command with arguments, capturing stdout.
func (c *Calibre) run(ctx context.Context, command string, args ...string) (string, error) {
	if c.override != "" {
		return c.override, nil
	}

	buf := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, command)
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

// Run calibredb with the given arguments, returning stdout.
func (c *Calibre) RunDBCommand(ctx context.Context, args ...string) (string, error) {
	if c.library != "" {
		args = append([]string{fmt.Sprintf("--library-path=%s", c.library)}, args...)
	}
	return c.run(ctx, "calibredb", args...)
}

// findFile attempts to locate the given target path within the base directory,
// where some ancestor of the target path has been replaced with the base
// directory.
func findFile(targetPath string, baseDir string) string {
	targetParts := strings.Split(filepath.ToSlash(filepath.Clean(targetPath)), "/")

	for i := len(targetParts) - 1; i >= 0; i-- {
		testPath := path.Join(append([]string{baseDir}, targetParts[i:]...)...)
		info, err := os.Stat(testPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			log.Printf("could not check %s: %v, ignoring", testPath, err)
		} else if info.IsDir() {
			//log.Printf("skipping directory %s", testPath)
			continue
		} else {
			return filepath.Clean(testPath)
		}
	}

	return ""
}

func (c *Calibre) GetBooks(ctx context.Context) ([]model.CalibreBook, error) {
	data, err := c.RunDBCommand(ctx, "list", "--for-machine", "--fields=all")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewBufferString(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return nil, fmt.Errorf("invalid leading token %s", delim)
	}
	var result []model.CalibreBook
	for decoder.More() {
		var next decodingBook
		if err = decoder.Decode(&next); err != nil {
			return nil, err
		}

		// Fix up the authors field:
		// "authors" can either be an array of string, or a bare string.  This
		// confuses the normal golang json decoder, so we have to create a
		// struct that serializes it as `any`, and then inspect the result to
		// set the result correctly.
		if next.Authors == nil {
			return nil, fmt.Errorf("could not find authors in %s", next.CalibreBook.Title)
		} else if authors, ok := next.Authors.([]any); ok {
			next.CalibreBook.Authors = make([]string, 0, len(authors))
			for _, authorObj := range authors {
				if author, ok := authorObj.(string); ok {
					next.CalibreBook.Authors = append(next.CalibreBook.Authors, author)
				} else {
					return nil, fmt.Errorf("could not parse %s: invalid author (%T) %v", next.CalibreBook.Title, authorObj, authorObj)
				}
			}
		} else if author, ok := next.Authors.(string); ok {
			next.CalibreBook.Authors = append(next.CalibreBook.Authors, author)
		} else {
			return nil, fmt.Errorf("could not parse %s: invalid authors (%T) %v", next.CalibreBook.Title, next.Authors, next.Authors)
		}

		// Fix up file paths:
		// The path stored in the database might have a different representation
		// for the library path (because we expect to run this in a docker
		// container).
		next.CalibreBook.Formats = util.Filter(
			util.Map(next.CalibreBook.Formats, func(path string) string {
				return findFile(path, c.library)
			}),
			func(path string) bool { return path != "" })
		next.CalibreBook.Cover = findFile(next.CalibreBook.Cover, c.library)

		result = append(result, *next.CalibreBook)
	}
	return result, nil
}
