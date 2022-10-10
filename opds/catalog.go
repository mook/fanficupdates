package opds

import (
	"encoding/xml"
	"time"

	"github.com/mook/fanficupdates/model"
)

type feedLink struct {
	Type string `xml:"type,attr"`
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Feed struct {
	XMLName xml.Name
	Title   string   `xml:"title"`
	Author  string   `xml:"author>name"`
	Id      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Start   feedLink `xml:"link"`
	Entries []*Entry
}

// MakeCatalog creates an OPDS catalog feed for the given books.
// Optionally, override the update date; if given an empty string, the current
// time is used.
func MakeCatalog(books []model.CalibreBook, updatedDate string) *Feed {
	if updatedDate == "" {
		updatedDate = time.Now().String()
	}
	result := &Feed{
		XMLName: xml.Name{Space: "http://www.w3.org/2005/Atom", Local: "feed"},
		Title:   "Library",
		Author:  "FanFicUpdates",
		Id:      "fanficupdates:all",
		Updated: updatedDate,
		Start: feedLink{
			Type: "application/atom+xml;type=feed;profile=opds-catalog",
			Rel:  "start",
			Href: "/opds",
		},
	}
	for _, book := range books {
		result.Entries = append(result.Entries, MakeEntry(book))
	}
	return result
}
