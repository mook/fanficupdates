package model_test

import (
	"net/url"
	"testing"

	"github.com/mook/fanficupdates/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUrl(t *testing.T) {
	t.Run("no url", func(t *testing.T) {
		book := model.CalibreBook{Identifiers: make(map[string]string)}
		assert.Nil(t, book.Url())
	})
	t.Run("invalid url", func(t *testing.T) {
		book := model.CalibreBook{
			Identifiers: map[string]string{
				"url": "://",
			},
		}
		assert.Nil(t, book.Url())
	})
	t.Run("valid url", func(t *testing.T) {
		expected, err := url.Parse("http://www.example.com/book?id=1")
		require.NoError(t, err)
		book := model.CalibreBook{
			Identifiers: map[string]string{
				"url": expected.String(),
			},
		}
		assert.Equal(t, expected, book.Url())
	})
}

func TestFilePath(t *testing.T) {
	t.Run("has epub", func(t *testing.T) {
		book := model.CalibreBook{
			Formats: []string{
				"/path/to/ignored.pdf",
				"/path/to/wanted.epub",
				"/path/to/other.epub",
			},
		}
		assert.Equal(t, "/path/to/wanted.epub", book.FilePath())
	})
	t.Run("missing epub", func(t *testing.T) {
		book := model.CalibreBook{
			Formats: []string{
				"/path/to/ignored.pdf",
			},
		}
		assert.Empty(t, book.FilePath())
	})
}
