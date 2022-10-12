package calibre

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
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

func TestRunDBCommand(t *testing.T) {
	subject := Calibre{}
	output, err := subject.RunDBCommand(context.Background(), "list", "--help")
	require.NoError(t, err)
	// Look for the "Created by Kovid Goyal" line, near the end.
	require.Contains(t, output, "Created by")
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
		Timestamp:    model.Time3339{Time: util.RandomTime()},
		PubDate:      model.Time3339{Time: util.RandomTime()},
		LastModified: model.Time3339{Time: util.RandomTime()},
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
	subject := (&Calibre{}).WithOverride(buf.String())
	books, err := subject.GetBooks(context.Background())
	require.NoError(t, err)
	require.Len(t, books, 1, "Unexpected number of books")
	actual := books[0]
	assert.Equal(t, expected, actual)
}

func TestGetBooksSingleAuthor(t *testing.T) {
	input := `[{"id":5,"authors":"Single Author"}]`
	subject := (&Calibre{}).WithOverride(input)
	books, err := subject.GetBooks(context.Background())
	require.NoError(t, err)
	require.Len(t, books, 1)
	book := books[0]
	require.Equal(t, []string{"Single Author"}, book.Authors)
}
