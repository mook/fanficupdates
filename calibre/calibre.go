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
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/sirupsen/logrus"
)

type Calibre struct {
	Library  string // Path to the Calibre library
	Settings string // Path to the settings directory

	// RunShim is used to mock running actual executables.  This should not be
	// used normally.
	RunShim func(cmd *exec.Cmd) ([]byte, error)
}

type decodingBook struct {
	*model.CalibreBook
	Authors any // Override author field; see GetBooks() for details.
}

// FindPaths attempts to auto-detect the Settings path and the Library path, if
// either were not specified.
func (c *Calibre) FindPaths(ctx context.Context) error {
	if c.Settings == "" {
		script := "import calibre.constants; print(calibre.config_dir)"
		output, err := c.Run(ctx, "calibre-debug", "--command", script)
		if err != nil {
			return fmt.Errorf("could not find settings path: %w", err)
		}
		c.Settings = filepath.Clean(strings.TrimSpace(output))
	}
	if c.Library == "" {
		script := "import calibre.library; print(calibre.library.current_library_path())"
		output, err := c.Run(ctx, "calibre-debug", "--command", script)
		if err != nil {
			return fmt.Errorf("could not find library path: %w", err)
		}
		output = strings.TrimSpace(output)
		if runtime.GOOS == "windows" && strings.HasPrefix(output, "//") {
			// On Windows, Calibre uses double-slash for UNC paths
			output = filepath.FromSlash(output)
		}
		logrus.Debugf("auto-detecting library path %s", output)
		c.Library = filepath.Clean(output)
	}
	return nil
}

// Run the given command with arguments, capturing stdout.
func (c *Calibre) Run(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command)
	cmd.Args = append(cmd.Args, args...)
	cmd.Stderr = os.Stderr
	if c.Settings != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CALIBRE_CONFIG_DIRECTORY=%s", c.Settings))
	}
	var buf []byte
	var err error
	if c.RunShim != nil {
		buf, err = c.RunShim(cmd)
	} else {
		buf, err = cmd.Output()
	}
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// Run calibredb with the given arguments, returning stdout.
func (c *Calibre) runDBCommand(ctx context.Context, args ...string) (string, error) {
	if c.Library != "" {
		args = append([]string{fmt.Sprintf("--library-path=%s", c.Library)}, args...)
	}
	return c.Run(ctx, "calibredb", args...)
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
			logrus.Debugf("skipping directory %s", testPath)
		} else {
			return filepath.Clean(testPath)
		}
	}

	logrus.Debugf("could not find file %s from %s", targetPath, baseDir)
	return ""
}

func (c *Calibre) GetBooks(ctx context.Context) ([]model.CalibreBook, error) {
	data, err := c.runDBCommand(ctx, "list", "--for-machine", "--fields=all")
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
				return findFile(path, c.Library)
			}),
			func(path string) bool { return path != "" })
		next.CalibreBook.Cover = findFile(next.CalibreBook.Cover, c.Library)

		result = append(result, *next.CalibreBook)
	}
	return result, nil
}

type UpdateMeta struct {
	Authors   []string  `calibre:"authors"`
	Comments  string    `calibre:"comments"`
	Published time.Time `calibre:"pubdate"`
	Publisher string    `calibre:"publisher"`
	Series    string    `calibre:"series"`
	Timestamp time.Time `calibre:"timestamp"`
}

func serializeMetadata(value reflect.Value) (string, error) {
	switch value.Kind() {
	case reflect.Array, reflect.Slice:
		var result []string
		for i := 0; i < value.Len(); i++ {
			next, err := serializeMetadata(value.Index(i))
			if err != nil {
				return "", err
			}
			result = append(result, next)
		}
		return strings.Join(result, ","), nil
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint()), nil
	case reflect.Interface, reflect.Pointer:
		return serializeMetadata(value.Elem())
	case reflect.Struct:
		time3339 := reflect.TypeOf(model.Time3339{})
		if value.CanConvert(time3339) {
			converted := value.Convert(time3339)
			if converted.IsZero() {
				return "", nil
			}
			return converted.Interface().(model.Time3339).Format(time.RFC3339), nil
		}
		stdTime := reflect.TypeOf(time.Time{})
		if value.CanConvert(stdTime) {
			converted := value.Convert(stdTime)
			if converted.IsZero() {
				return "", nil
			}
			return converted.Interface().(time.Time).Format(time.RFC3339), nil
		}
		return "", fmt.Errorf("don't know how to serialize %s/%s", value.Type().PkgPath(), value.Type().Name())
	}
	return "", fmt.Errorf("don't know how to serialize %s", value.Kind())
}

func (c *Calibre) UpdateBook(ctx context.Context, id int, meta UpdateMeta) error {
	args := []string{"set_metadata"}
	val := reflect.ValueOf(meta)
	for i := 0; i < val.Type().NumField(); i++ {
		tag := val.Type().Field(i).Tag.Get("calibre")
		value, err := serializeMetadata(val.Field(i))
		if err != nil {
			return err
		}
		if value != "" {
			args = append(args, fmt.Sprintf("--field=%s:%s", tag, value))
		}
	}
	args = append(args, fmt.Sprintf("%d", id))
	_, err := c.runDBCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("could not update database for book #%d: %w", id, err)
	}
	return nil
}
