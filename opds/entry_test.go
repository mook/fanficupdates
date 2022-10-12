package opds

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math/rand"
	"testing"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/require"
)

func makeBook(t *testing.T) *model.CalibreBook {
	bookUuid, err := uuid.NewRandom()
	require.NoError(t, err, "error generating uuid")
	return &model.CalibreBook{
		Id:        rand.Int(),
		Uuid:      bookUuid.String(),
		Publisher: util.RandomString(),
		Size:      rand.Int(),
		Identifiers: map[string]string{
			"url": fmt.Sprintf("http://test1.com/?sid=%d", rand.Int()),
		},
		Formats: []string{
			fmt.Sprintf("/dev/does/not/exist/%s.epub", util.RandomString()),
		},
		Title:        util.RandomString(),
		Authors:      util.RandomList(5, util.RandomString),
		AuthorSort:   util.RandomString(),
		Timestamp:    model.Time3339{Time: util.RandomTime()},
		PubDate:      model.Time3339{Time: util.RandomTime()},
		LastModified: model.Time3339{Time: util.RandomTime()},
		Tags:         util.RandomList(5, util.RandomString),
		Comments:     util.RandomString(),
		Languages:    util.RandomList(5, util.RandomString),
		Cover:        util.RandomString(),
	}
}

func TestMakeEntry(t *testing.T) {
	book := makeBook(t)
	entry := MakeEntry(*book)
	rawActual, err := xml.MarshalIndent(entry, "", "  ")
	require.NoError(t, err, "error marshaling entry")
	prettyActual, err := util.PrettyXML(rawActual)
	require.NoError(t, err, "error pretty-printing actual output")

	tmpl := template.New("expected").Funcs(sprig.FuncMap())
	expected := bytes.Buffer{}
	expectedTemplate := `<entry xmlns="http://www.w3.org/2005/Atom">
		<title>{{.Book.Title}}</title>
		<author>{{range .Book.Authors}}<name>{{.}}</name>{{end}}</author>
		<id>urn:uuid:{{.Book.Uuid}}</id>
		<updated>{{.Book.LastModified.Format .RFC3339}}</updated>
		<published>{{.Book.Timestamp.Format .RFC3339}}</published>
		<date xmlns="http://purl.org/dc/terms/">{{.Book.PubDate.Format .RFC3339}}</date>
		<content type="xhtml">
			<div xmlns="http://www.w3.org/1999/xhtml">
				TAGS: {{ .Book.Tags | sortAlpha | join ", " }}
				<br />
				{{ .Book.Comments }}
			</div>
		</content>
		<link type="application/epub+zip"
			href="/get/epub/{{.Book.Id}}"
			rel="http://opds-spec.org/acquisition"
			length="{{.Book.Size}}"
			mtime="{{.Book.LastModified.Format .RFC3339}}" />
		<link type="image/jpeg" href="/get/cover/{{.Book.Id}}" rel="http://opds-spec.org/cover" />
		<link type="image/jpeg" href="/get/thumb/{{.Book.Id}}" rel="http://opds-spec.org/thumbnail" />
		<link type="image/jpeg" href="/get/cover/{{.Book.Id}}" rel="http://opds-spec.org/image" />
		<link type="image/jpeg" href="/get/thumb/{{.Book.Id}}" rel="http://opds-spec.org/image/thumbnail" />
	</entry>`
	template.Must(tmpl.Parse(expectedTemplate))
	require.NoError(t, tmpl.Execute(&expected, map[string]any{
		"Book":    book,
		"RFC3339": time.RFC3339,
	}), "error rendering expected output")
	prettyExpected, err := util.PrettyXML(expected.Bytes())
	require.NoError(t, err, "error pretty-printing expected output")
	require.Equal(t, string(prettyExpected), string(prettyActual))
}
