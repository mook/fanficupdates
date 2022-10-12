package opds

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalog(t *testing.T) {
	subject := NewServer()
	server := httptest.NewServer(subject.Handler)
	defer server.Close()

	res, err := http.Get(fmt.Sprintf("%s/opds", server.URL))
	require.NoError(t, err)
	rawActual, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	prettyActual, err := util.PrettyXML(rawActual)
	require.NoError(t, err, "error pretty-printing actual output")

	// Actually testing that the catalog is valid is done in catalog_test.go
	// This just checks that the feed looks reasonable.
	assert.Contains(t, string(prettyActual), "<feed")
}

func TestDownload(t *testing.T) {
	subject := NewServer()
	subject.Books = util.RandomList(5, func() model.CalibreBook { return *makeBook(t) })
	subject.Books = append(subject.Books, *makeBook(t)) // At least one book
	server := httptest.NewServer(subject.Handler)
	defer server.Close()

	t.Run("invalid request", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/epub/", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Invalid path")
		}
	})
	t.Run("non-numeric id", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/epub/pika", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Failed to convert")
		}
	})
	t.Run("missing id", func(t *testing.T) {
		var id int
		for id = 0; id <= len(subject.Books); id++ {
			hasId := util.Any(subject.Books, func(book model.CalibreBook) bool {
				return book.Id == id
			})
			if !hasId {
				break
			}
		}

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Could not find book")
		}
	})

	t.Run("missing formats", func(t *testing.T) {
		book := &subject.Books[0]
		formats := book.Formats
		book.Formats = []string{"something.arj"}
		defer func() { book.Formats = formats }()

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Could not find epub")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		book := &subject.Books[0]
		formats := book.Formats
		defer func() { book.Formats = formats }()
		workdir := t.TempDir()
		workpath := path.Join(workdir, "hello.epub")
		book.Formats = []string{workpath}

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Missing epub")
		}
	})

	t.Run("valid format", func(t *testing.T) {
		book := &subject.Books[0]
		formats := book.Formats
		defer func() { book.Formats = formats }()
		workdir := t.TempDir()
		workpath := path.Join(workdir, "hello.epub")
		book.Formats = []string{workpath}
		expected := "pikachu"
		require.NoError(t, os.WriteFile(workpath, []byte(expected), 0o755))

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
	})
}
