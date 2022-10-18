package model

import (
	"net/url"
	"strings"
)

type CalibreBook struct {
	Id           int
	Uuid         string
	Publisher    string
	Size         int
	Identifiers  map[string]string
	Formats      []string
	Title        string
	Authors      []string
	AuthorSort   string `json:"author_sort"`
	Timestamp    Time3339
	PubDate      Time3339
	LastModified Time3339 `json:"last_modified"`
	Tags         []string
	Comments     string
	Languages    []string
	Cover        string
	SeriesIndex  *float64 `json:"series_index"`
}

// Url returns the source URL for the book, or nil if unavailable.
func (b *CalibreBook) Url() *url.URL {
	spec, ok := b.Identifiers["url"]
	if !ok {
		return nil
	}
	u, err := url.Parse(spec)
	if err != nil {
		return nil
	}
	return u
}

// FilePath returns the path to the epub for the book, or empty string if not
// found.
func (b *CalibreBook) FilePath() string {
	for _, path := range b.Formats {
		if strings.HasSuffix(path, ".epub") {
			return path
		}
	}
	return ""
}
