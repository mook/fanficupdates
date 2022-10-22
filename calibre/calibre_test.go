package calibre

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindPaths(t *testing.T) {
	type entry struct {
		script string
		result string
	}
	expectedScripts := []entry{
		{
			script: "import calibre.constants; print(calibre.config_dir)",
			result: "/path/to/settings/directory\n",
		},
		{
			script: "import calibre.library; print(calibre.library.current_library_path())",
			result: "  /path//to/settings/../library/directory      \n",
		},
	}
	runCount := 0
	testContext := context.Background()
	subject := Calibre{RunShim: func(cmd *exec.Cmd) ([]byte, error) {
		assert.Contains(
			t,
			[]string{"calibre-debug", "calibre-debug.exe"},
			filepath.Base(cmd.Path),
			fmt.Errorf("incorrect executable on call %d", runCount))
		if assert.Less(t, runCount, len(expectedScripts), fmt.Sprintf("called too many times (got %d)", runCount)) {
			assert.Equal(
				t,
				[]string{
					"calibre-debug",
					"--command",
					expectedScripts[runCount].script,
				},
				cmd.Args,
				fmt.Sprintf("incorrect arguments on call %d", runCount))
			script := expectedScripts[runCount].result
			runCount++
			return []byte(script), nil
		}
		return nil, fmt.Errorf("too many calls")

	}}
	assert.NoError(t, subject.FindPaths(testContext))
	assert.Equal(t, filepath.Clean("/path/to/settings/directory"), subject.Settings, "incorrect settings directory")
	assert.Equal(t, filepath.Clean("/path/to/library/directory"), subject.Library, "incorrect library directory")
}

func TestFindPathsError(t *testing.T) {
	t.Run("settings", func(t *testing.T) {
		targetError := fmt.Errorf("some error")
		c := &Calibre{
			RunShim: func(cmd *exec.Cmd) ([]byte, error) {
				return nil, targetError
			},
		}
		err := c.FindPaths(context.Background())
		assert.ErrorContains(t, err, "could not find settings path")
		assert.ErrorIs(t, err, targetError)
	})
	t.Run("library", func(t *testing.T) {
		targetError := fmt.Errorf("some error")
		c := &Calibre{
			Settings: "something",
			RunShim: func(cmd *exec.Cmd) ([]byte, error) {
				assert.Contains(t, cmd.Env, "CALIBRE_CONFIG_DIRECTORY=something")
				return nil, targetError
			},
		}
		err := c.FindPaths(context.Background())
		assert.ErrorContains(t, err, "could not find library path")
		assert.ErrorIs(t, err, targetError)
	})
}

func TestRunDBCommand(t *testing.T) {
	t.Run("captures calibre output", func(t *testing.T) {
		subject := Calibre{}
		output, err := subject.runDBCommand(context.Background(), "list", "--help")
		if assert.NoError(t, err) {
			// Look for the "Created by Kovid Goyal" line, near the end.
			assert.Contains(t, output, "Created by")
		}
	})
	t.Run("sets library path", func(t *testing.T) {
		expected := "sample output"
		c := Calibre{
			Library: "/path/to/a/library",
			RunShim: func(cmd *exec.Cmd) ([]byte, error) {
				assert.Equal(t,
					[]string{"calibredb", "--library-path=/path/to/a/library", "some", "args"},
					cmd.Args)
				assert.Nil(t, cmd.Stdout)
				return []byte(expected), nil
			},
		}
		result, err := c.runDBCommand(context.Background(), "some", "args")
		if assert.NoError(t, err) {
			assert.Equal(t, expected, result)
		}
	})
}

func TestFindFile(t *testing.T) {
	workdir := t.TempDir()
	parts := []string{"one", "two", "three", "file"}
	desiredPath := path.Join(append([]string{workdir}, parts...)...)
	require.NoError(t, os.MkdirAll(path.Dir(desiredPath), 0755))
	require.NoError(t, os.Truncate(desiredPath, 0))

	destdir := t.TempDir()
	targetPath := path.Join(append([]string{destdir}, parts...)...)

	result := findFile(targetPath, workdir)
	assert.Equal(t, filepath.Clean(desiredPath), filepath.Clean(result))
}

func TestGetBooks(t *testing.T) {
	workdir := t.TempDir()
	bookPath := filepath.Clean(path.Join(workdir, "test.epub"))
	require.NoError(t, os.WriteFile(bookPath, []byte("this is a book"), 0755))
	coverPath := filepath.Clean(path.Join(workdir, util.RandomString()+".jpg"))
	require.NoError(t, os.WriteFile(coverPath, []byte("!cover"), 0755))
	expected := model.CalibreBook{
		Id:        rand.Int(),
		Uuid:      uuid.NewString(),
		Publisher: util.RandomString(),
		Size:      rand.Int(),
		Identifiers: map[string]string{
			"url": fmt.Sprintf("http://test1.com/?sid=%d", rand.Int()),
		},
		Formats:      []string{bookPath},
		Title:        util.RandomString(),
		Authors:      util.RandomList(5, util.RandomString),
		AuthorSort:   util.RandomString(),
		Timestamp:    *model.NewTime3339(util.RandomTime()),
		PubDate:      *model.NewTime3339(util.RandomTime()),
		LastModified: *model.NewTime3339(util.RandomTime()),
		Tags:         util.RandomList(5, util.RandomString),
		Comments:     util.RandomString(),
		Languages:    util.RandomList(5, util.RandomString),
		Cover:        coverPath,
	}
	tmpl, err := template.New("input").Funcs(sprig.FuncMap()).Parse(`[{
		"id": {{ .Book.Id }},
		"uuid": "{{ .Book.Uuid }}",
		"publisher": "{{ .Book.Publisher }}",
		"size": {{.Book.Size}},
		"identifiers": {{ .Book.Identifiers | toJson }},
		"formats": {{ .Book.Formats | toJson }},
		"title": "{{ .Book.Title }}",
		"authors": {{ .Book.Authors | toJson }},
		"author_sort": "{{ .Book.AuthorSort }}",
		"timestamp": "{{ .Book.Timestamp.Format .RFC3339 }}",
		"pubdate": "{{ .Book.PubDate.Format .RFC3339 }}",
		"last_modified": "{{ .Book.LastModified.Format .RFC3339 }}",
		"tags": {{ .Book.Tags | toJson }},
		"comments": "{{ .Book.Comments }}",
		"languages": {{ .Book.Languages | toJson }},
		"cover": {{ .Book.Cover | toJson }}
	}]`)
	require.NoError(t, err, "could not parse template")
	buf := &bytes.Buffer{}
	require.NoError(t, tmpl.Execute(buf, map[string]any{
		"Book":    expected,
		"RFC3339": time.RFC3339,
	}))
	subject := &Calibre{
		RunShim: func(cmd *exec.Cmd) ([]byte, error) {
			assert.Nil(t, cmd.Stdout)
			return buf.Bytes(), nil
		},
	}
	books, err := subject.GetBooks(context.Background())
	require.NoError(t, err)
	require.Len(t, books, 1, "Unexpected number of books")
	actual := books[0]
	assert.Equal(t, expected, actual)
}

func TestGetBooksSingleAuthor(t *testing.T) {
	input := `[{"id":5,"authors":"Single Author"}]`
	subject := &Calibre{
		RunShim: func(cmd *exec.Cmd) ([]byte, error) {
			assert.Nil(t, cmd.Stdout)
			return []byte(input), nil
		},
	}
	books, err := subject.GetBooks(context.Background())
	require.NoError(t, err)
	require.Len(t, books, 1)
	book := books[0]
	require.Equal(t, []string{"Single Author"}, book.Authors)
}

func TestUpdateBook(t *testing.T) {
	type testCase struct {
		name string
		UpdateMeta
		args   []string
		result error
	}

	testCases := []testCase{
		{
			name: "filled",
			UpdateMeta: UpdateMeta{
				Authors:   []string{"foo", "bar"},
				Comments:  "some comment",
				Published: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
				Publisher: "hydraulic press",
				Series:    "112358",
				Timestamp: time.Date(1234, 5, 6, 7, 8, 9, 0, time.UTC),
			},
			args: []string{
				"--field=authors:foo,bar",
				"--field=comments:some comment",
				"--field=pubdate:2006-01-02T15:04:05Z",
				"--field=publisher:hydraulic press",
				"--field=series:112358",
				"--field=timestamp:1234-05-06T07:08:09Z",
			},
		},
		{
			name: "empty",
		},
		{
			name:   "error",
			result: fmt.Errorf("some error"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var args []string
			c := Calibre{
				RunShim: func(cmd *exec.Cmd) ([]byte, error) {
					args = cmd.Args
					return nil, testCase.result
				},
			}
			err := c.UpdateBook(
				context.Background(),
				12345,
				testCase.UpdateMeta,
			)
			if testCase.result == nil {
				assert.NoError(t, err)
				expected := append([]string{"calibredb", "set_metadata"}, testCase.args...)
				expected = append(expected, "12345")
				assert.Equal(t, expected, args)
			} else {
				assert.ErrorIs(t, err, testCase.result)
			}
		})
	}
}
