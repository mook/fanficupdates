package util

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

type XMLElement struct {
	XMLName  xml.Name
	Attr     []xml.Attr   `xml:",any,attr"`
	Children []XMLElement `xml:",any"`
	Text     string       `xml:",chardata"`
	Comment  string       `xml:",comment"`
}

func PrettyXML(input []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	decoder := xml.NewDecoder(bytes.NewReader(input))
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			encoder.Flush()
			return buf.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
		if elem, ok := token.(xml.StartElement); ok {
			// Remove duplicate xmlns= attributes
			offset := 0
			for i, attr := range elem.Attr[:] {
				if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
					elem.Name.Space = attr.Value
					elem.Attr = append(elem.Attr[:i+offset], elem.Attr[i+offset+1:]...)
					offset += 1
				}
			}
			token = elem
		}
		if data, ok := token.(xml.CharData); ok {
			// Trim whitespace on any character data
			token = xml.CharData(strings.TrimSpace(string(data)))
		}
		if err = encoder.EncodeToken(token); err != nil {
			return nil, err
		}
	}
}
