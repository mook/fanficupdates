package fanficfare

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/mook/fanficupdates/calibre"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util/assertx"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFanFicFare(t *testing.T) {
	t.Run("fail to list sites", func(t *testing.T) {
		expectedError := fmt.Errorf("failed to list sites")
		c := calibre.Calibre{
			RunShim: func(cmd *exec.Cmd) ([]byte, error) {
				assert.Equal(t, []string{
					"calibre-debug", "--run-plugin=FanFicFare", "--",
					"--non-interactive", "--library-path=somewhere",
					"--sites-list",
				}, cmd.Args)
				return nil, expectedError
			},
			Library: "somewhere",
		}

		_, err := NewFanFicFare(context.Background(), &c)
		assert.ErrorIs(t, err, expectedError)
	})
	t.Run("reads site list", func(t *testing.T) {
		type site struct {
			name     string
			tld      string
			examples []string
		}
		sites := []site{
			{
				name:     "Alpha",
				tld:      "alpha.test",
				examples: []string{"http://alpha.test/1234"},
			},
			{
				name: "Beta",
				tld:  "beta.test",
				examples: []string{
					"http://beta.test/1234",
					"https://subdomain.beta.test/1234",
				},
			},
			{
				name: "Gamma",
				tld:  "gamma.test",
				examples: []string{
					"https://gamma.test/1234",
					"https://gamma.test/story?id=1234&chapter=9",
				},
			},
			{
				name:     "Invalid",
				tld:      "",
				examples: []string{"no", "valid", "examples"},
			},
		}
		supportedSites := map[string]struct{}{}
		c := calibre.Calibre{
			RunShim: func(cmd *exec.Cmd) ([]byte, error) {
				var buf bytes.Buffer
				for _, site := range sites {
					buf.WriteString(fmt.Sprintf("#### %s\n", site.name))
					buf.WriteString("Example URLs:\n")
					for _, example := range site.examples {
						buf.WriteString(fmt.Sprintf("  * %s\n", example))
					}
					if site.tld != "" {
						supportedSites[site.tld] = struct{}{}
					}
				}
				return buf.Bytes(), nil
			},
		}
		ff, err := NewFanFicFare(context.Background(), &c)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, ff.supportedSites)
			assert.Equal(t, supportedSites, ff.supportedSites)
		}
	})
}

func TestProcess(t *testing.T) {
	makeFff := func() (*FanFicFare, *test.Hook) {
		logger, hook := test.NewNullLogger()
		fff := &FanFicFare{
			calibre: &calibre.Calibre{},
			supportedSites: map[string]struct{}{
				"supported.test": {},
			},
			logger: *logger,
		}
		return fff, hook
	}
	makeBook := func(url string) model.CalibreBook {
		return model.CalibreBook{
			Title: "Sample Book",
			Identifiers: map[string]string{
				"url": url,
			},
		}
	}
	t.Run("no url", func(t *testing.T) {
		subj, hook := makeFff()
		ok, err := subj.Process(context.Background(), model.CalibreBook{})
		assert.NoError(t, err)
		assert.False(t, ok)
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return strings.Contains(entry.Message, "no URL")
		})
	})
	t.Run("invalid url", func(t *testing.T) {
		subj, hook := makeFff()
		book := makeBook("http:///path")
		_, err := subj.Process(context.Background(), book)
		assert.Error(t, err)
		assert.Empty(t, hook.AllEntries())
	})
	t.Run("unsupported site", func(t *testing.T) {
		subj, hook := makeFff()
		book := makeBook("http://unsupported.test/")
		ok, err := subj.Process(context.Background(), book)
		assert.NoError(t, err)
		assert.False(t, ok)
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return strings.Contains(entry.Message, "not supported")
		})
	})
	t.Run("no changes required", func(t *testing.T) {
		file, err := os.Create(path.Join(t.TempDir(), "test.epub"))
		require.NoError(t, err)
		file.Close()
		subj, hook := makeFff()
		message := "Not updating Sample Book"
		book := makeBook("http://supported.test")
		book.Formats = append(book.Formats, file.Name())
		subj.calibre.RunShim = func(cmd *exec.Cmd) ([]byte, error) {
			assert.Contains(t, cmd.Args, "--json-meta")
			assert.Contains(t, cmd.Args, "--update-epub")
			// Expected message, plus JSON data - should be whole thing, but
			// we can get away with nothing.
			output := message + "\n{\n}\n"
			return []byte(output), nil
		}
		ok, err := subj.Process(context.Background(), book)
		assert.NoError(t, err)
		assert.False(t, ok)
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return strings.Contains(entry.Message, "Updating Sample Book")
		})
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return entry.Message == message
		}, "expected message: %s", message)
	})
	t.Run("fail to update metadata", func(t *testing.T) {
		file, err := os.Create(path.Join(t.TempDir(), "test.epub"))
		require.NoError(t, err)
		file.Close()
		subj, hook := makeFff()
		book := makeBook("http://supported.test")
		message := fmt.Sprintf("Do update - %s", book.Title)
		book.Formats = append(book.Formats, file.Name())
		runCount := 0
		subj.calibre.RunShim = func(cmd *exec.Cmd) ([]byte, error) {
			if runCount == 0 {
				runCount++
				assert.Contains(t, cmd.Args, "--json-meta")
				assert.Contains(t, cmd.Args, "--update-epub")
				output := message + "\n{\n\"author\": \"someone\"}"
				return []byte(output), nil
			} else if runCount == 1 {
				runCount++
				expected := []string{
					"calibredb",
					"set_metadata",
					"--field=authors:someone",
					fmt.Sprintf("%d", book.Id),
				}
				assert.Equal(t, expected, cmd.Args)
				return []byte{}, fmt.Errorf("some error")
			}
			assert.Fail(
				t,
				"invalid run count",
				"got unexpected run count %d with command %#v", runCount, cmd.Args)
			return nil, fmt.Errorf("running executables too many times")
		}
		_, err = subj.Process(context.Background(), book)
		assert.Error(t, err)
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return strings.Contains(entry.Message, "Updating Sample Book")
		})
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return entry.Message == message
		}, "expected message: %s", message)
	})
	t.Run("successfully update book ", func(t *testing.T) {
		file, err := os.Create(path.Join(t.TempDir(), "test.epub"))
		require.NoError(t, err)
		file.Close()
		subj, hook := makeFff()
		book := makeBook("http://supported.test")
		message := fmt.Sprintf("Do update - %s", book.Title)
		book.Formats = append(book.Formats, file.Name())
		runCount := 0
		subj.calibre.RunShim = func(cmd *exec.Cmd) ([]byte, error) {
			if runCount == 0 {
				runCount++
				assert.Contains(t, cmd.Args, "--json-meta")
				assert.Contains(t, cmd.Args, "--update-epub")
				output := message + "\n{\n\"author\": \"someone\"}"
				return []byte(output), nil
			} else if runCount == 1 {
				runCount++
				expected := []string{
					"calibredb",
					"set_metadata",
					"--field=authors:someone",
					fmt.Sprintf("%d", book.Id),
				}
				assert.Equal(t, expected, cmd.Args)
				return []byte{}, nil
			}
			assert.Fail(
				t,
				"invalid run count",
				"got unexpected run count %d with command %#v", runCount, cmd.Args)
			return nil, fmt.Errorf("running executables too many times")
		}
		ok, err := subj.Process(context.Background(), book)
		assert.NoError(t, err)
		assert.True(t, ok)
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return strings.Contains(entry.Message, "Updating Sample Book")
		})
		assertx.Any(t, hook.AllEntries(), func(entry *logrus.Entry) bool {
			return entry.Message == message
		}, "expected message: %s", message)
	})
}
