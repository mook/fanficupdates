package opds

import (
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
)

type entryLink struct {
	Type   string `xml:"type,attr"`
	Href   string `xml:"href,attr"`
	Rel    string `xml:"rel,attr"`
	Length int    `xml:"length,attr,omitempty"`
	MTime  string `xml:"mtime,attr,omitempty"`
}

type Entry struct {
	XMLName      xml.Name
	Title        string   `xml:"title"`
	Author       []string `xml:"author>name"`
	Id           string   `xml:"id"`
	LastModified string   `xml:"updated"`
	Timestamp    string   `xml:"published"`
	PubDate      struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	} `xml:"date"`
	Content struct {
		Type           string `xml:"type,attr"`
		ContentWrapper struct {
			XMLName   xml.Name
			Tags      string `xml:",chardata"`
			LineBreak string `xml:"br"`
			Comments  string `xml:",innerxml"`
		}
	} `xml:"content"`
	Links []entryLink `xml:"link"`
}

func MakeEntry(book model.CalibreBook) *Entry {
	entry := &Entry{
		XMLName: xml.Name{
			Local: "entry",
			Space: "http://www.w3.org/2005/Atom",
		},
		Title:        book.Title,
		Author:       book.Authors[:],
		Id:           fmt.Sprintf("urn:uuid:%s", book.Uuid),
		LastModified: book.LastModified.Format(time.RFC3339),
		Timestamp:    book.Timestamp.Format(time.RFC3339),
	}
	entry.PubDate.XMLName = xml.Name{Space: "http://purl.org/dc/terms/", Local: "date"}
	entry.PubDate.Value = book.PubDate.Format(time.RFC3339)
	entry.Content.Type = "xhtml"
	entry.Content.ContentWrapper.XMLName.Space = "http://www.w3.org/1999/xhtml"
	entry.Content.ContentWrapper.XMLName.Local = "div"
	hasEpub := util.Any(book.Formats, func(f string) bool { return path.Ext(f) == ".epub" })
	if hasEpub {
		entry.Links = append(entry.Links, entryLink{
			Type:   "application/epub+zip",
			Href:   fmt.Sprintf("/get/epub/%d", book.Id),
			Rel:    "http://opds-spec.org/acquisition",
			Length: book.Size,
			MTime:  book.LastModified.Format(time.RFC3339),
		})
	}
	link := func(kind string, id int, rel string) entryLink {
		return entryLink{
			Type: "image/jpeg",
			Href: fmt.Sprintf("/get/%s/%d", kind, id),
			Rel:  fmt.Sprintf("http://opds-spec.org/%s", rel),
		}
	}
	entry.Links = append(entry.Links, link("cover", book.Id, "cover"))
	entry.Links = append(entry.Links, link("thumb", book.Id, "thumbnail"))
	entry.Links = append(entry.Links, link("cover", book.Id, "image"))
	entry.Links = append(entry.Links, link("thumb", book.Id, "image/thumbnail"))
	tags := book.Tags[:]
	sort.Strings(tags)
	entry.Content.ContentWrapper.Tags = fmt.Sprintf("TAGS: %s", strings.Join(tags, ", "))
	entry.Content.ContentWrapper.Comments = book.Comments
	return entry
}
