package opds

import (
	"bytes"
	"encoding/xml"
	"testing"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/require"
)

func TestMakeCatalog(t *testing.T) {
	var books []model.CalibreBook
	updatedDate := time.Now().String()

	for i := 0; i < 5; i++ {
		books = append(books, *makeBook(t))
	}
	catalog := MakeCatalog(books, updatedDate)
	rawActual, err := xml.Marshal(catalog)
	require.NoError(t, err, "error marshaling catalog")
	prettyActual, err := util.PrettyXML(rawActual)
	require.NoError(t, err, "error pretty-printing actual output")

	tmpl := template.New("expected").Funcs(sprig.FuncMap())
	expected := bytes.Buffer{}
	expectedTemplate := `<feed xmlns="http://www.w3.org/2005/Atom">
		<title>Library</title>
		<author><name>FanFicUpdates</name></author>
		<id>fanficupdates:all</id>
		<updated>{{ .UpdatedDate }}</updated>
		<link type="application/atom+xml;type=feed;profile=opds-catalog" rel="start" href="/opds" />
		{{ range .Entries}} {{ . }} {{ end }}
	</feed>`
	template.Must(tmpl.Parse(expectedTemplate))
	require.NoError(t, tmpl.Execute(&expected, map[string]any{
		"UpdatedDate": updatedDate,
		"Entries": util.Map(books, func(book model.CalibreBook) string {
			entryXML, err := xml.Marshal(MakeEntry(book))
			require.NoError(t, err)
			return string(entryXML)
		}),
	}), "error rendering expected output")
	prettyExpected, err := util.PrettyXML(expected.Bytes())
	require.NoError(t, err, "error pretty-printing expected output")
	require.Equal(t, string(prettyExpected), string(prettyActual))
}
