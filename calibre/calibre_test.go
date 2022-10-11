package calibre_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
	"github.com/mook/fanficupdates/calibre"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	subject := calibre.Calibre{}
	output, err := subject.Run(context.Background(), "list", "--help")
	require.NoError(t, err)
	// Look for the "Created by Kovid Goyal" line, near the end.
	require.Contains(t, output, "Created by")
}

func TestGetBooks(t *testing.T) {
	randString := util.NewRandomStringReader()
	expected := model.CalibreBook{
		Id:        rand.Int(),
		Uuid:      uuid.NewString(),
		Publisher: randString.MustReadString(32),
		Size:      rand.Int(),
		Identifiers: map[string]string{
			"url": fmt.Sprintf("http://test1.com/?sid=%d", rand.Int()),
		},
		// Formats: ,
		Title:        randString.MustReadString(32),
		Authors:      randString.MustReadStringSlice(5, 32),
		AuthorSort:   randString.MustReadString(32),
		Timestamp:    model.Time3339{Time: util.RandomTime()},
		PubDate:      model.Time3339{Time: util.RandomTime()},
		LastModified: model.Time3339{Time: util.RandomTime()},
		Tags:         randString.MustReadStringSlice(5, 32),
		Comments:     randString.MustReadString(32),
		Languages:    randString.MustReadStringSlice(5, 32),
		Cover:        randString.MustReadString(32),
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
		"cover": "{{ .Book.Cover }}"
	}]`)
	require.NoError(t, err, "could not parse template")
	buf := &bytes.Buffer{}
	require.NoError(t, tmpl.Execute(buf, map[string]any{
		"Book":    expected,
		"RFC3339": time.RFC3339,
	}))
	subject := (&calibre.Calibre{}).WithOverride(buf.String())
	books, err := subject.GetBooks(context.Background())
	require.NoError(t, err)
	require.Len(t, books, 1, "Unexpected number of books")
	actual := books[0]
	assert.Equal(t, expected, actual)
}

func TestGetBooksSingleAuthor(t *testing.T) {
	input := `[{"id":5,"authors":"Single Author"}]`
	subject := (&calibre.Calibre{}).WithOverride(input)
	books, err := subject.GetBooks(context.Background())
	require.NoError(t, err)
	require.Len(t, books, 1)
	book := books[0]
	require.Equal(t, []string{"Single Author"}, book.Authors)
}
