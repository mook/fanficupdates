package model

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
