package fanficfare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/mook/fanficupdates/calibre"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/sirupsen/logrus"
)

type meta struct {
	Author      string
	FormatName  string `json:"formatname"`
	Description string `json:"description"`
	LastUpdate  string `json:"lastupdate"`
	NumChapters string `json:"numChapters"`
	Publisher   string
	Published   model.Time3339 `json:"datePublished"`
	Series      string         `json:"series"`
	Site        string
	Status      string
	StoryURL    string `json:"storyUrl"`
	Title       string
	Updated     model.Time3339 `json:"dateUpdated"`
	Chapters    []chapter      `json:"zchapters"`
}

type chapterInner struct {
	Date   model.Time3339
	KWords string `json:"kwords"`
	Title  string
	URL    string `json:"url"`
	Words  string `json:"words"`
}

type chapter struct {
	Number int `json:"-"`
	chapterInner
}

func (c *chapter) UnmarshalJSON(data []byte) error {
	token, err := json.NewDecoder(bytes.NewBuffer(data[:])).Token()
	if err != nil {
		return err
	}
	if delim, ok := token.(json.Delim); !ok {
		return fmt.Errorf("unexpected json token %#v parsing chapters", token)
	} else if delim == '[' {
		parts := make([]json.RawMessage, 0, 2)
		err := json.Unmarshal(data, &parts)
		if err != nil {
			return err
		}
		if len(parts) != 2 {
			return fmt.Errorf("expected a two-tuple, got %d parts", len(parts))
		}
		if err = json.Unmarshal(parts[0], &c.Number); err != nil {
			return err
		}
		if err = json.Unmarshal(parts[1], &c.chapterInner); err != nil {
			return err
		}
	} else if delim == '{' {

	} else {
		return fmt.Errorf("unexpected json delimiter %#v parsing chapters", delim)
	}
	return nil
}

type FanFicFare struct {
	calibre        *calibre.Calibre
	supportedSites map[string]struct{}
	logger         logrus.Logger
}

func NewFanFicFare(ctx context.Context, calibre *calibre.Calibre) (*FanFicFare, error) {
	result := &FanFicFare{calibre: calibre, logger: *logrus.StandardLogger()}
	err := result.getSupportedSites(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Run FanFicFare with the given command, returning stdout.
func (f *FanFicFare) run(ctx context.Context, args ...string) (string, error) {
	resultArgs := []string{"--run-plugin=FanFicFare", "--", "--non-interactive"}
	if f.calibre.Library != "" {
		resultArgs = append(resultArgs, "--library-path="+f.calibre.Library)
	}
	resultArgs = append(resultArgs, args...)
	return f.calibre.Run(ctx, "calibre-debug", resultArgs...)
}

func (f *FanFicFare) getSupportedSites(ctx context.Context) error {
	f.supportedSites = make(map[string]struct{})
	matcher, err := regexp.Compile(`\s*\*\s*(\w+://[^/]+)`)
	if err != nil {
		return err
	}
	helpText, err := f.run(ctx, "--sites-list")
	if err != nil {
		return err
	}
	sites := make(map[string]struct{})
	for _, line := range strings.Split(helpText, "\n") {
		match := matcher.FindStringSubmatch(line)
		if match != nil {
			u, err := url.Parse(match[1])
			if err == nil {
				tld, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
				if err == nil {
					sites[tld] = struct{}{}
				}
			}
		}
	}
	f.supportedSites = sites
	return nil
}

// Process a single book, returning true if an update was found.
func (f *FanFicFare) Process(ctx context.Context, book model.CalibreBook) (bool, error) {
	url := book.Url()
	if url == nil {
		// Books without URL is just skipped without error.
		f.logger.Infof("Skipping %s, no URL", book.Title)
		return false, nil
	}

	tld, err := publicsuffix.EffectiveTLDPlusOne(url.Hostname())
	if err != nil {
		return false, fmt.Errorf("could not get eTLD for %s: %w", url, err)
	}

	if _, ok := f.supportedSites[tld]; !ok {
		f.logger.Infof("Skipping %s, not supported", url.String())
		return false, nil
	}

	f.logger.Infof("Updating %s: %s", book.Title, url)
	workFile, err := os.CreateTemp("", "fanficupdates-*.epub")
	if err != nil {
		return false, fmt.Errorf("could not create temporary file: %w", err)
	}
	defer os.Remove(workFile.Name())

	srcFile, err := os.Open(book.FilePath())
	if err != nil {
		return false, fmt.Errorf("could not open existing epub %s: %w", book.FilePath(), err)
	}
	if _, err = io.Copy(workFile, srcFile); err != nil {
		return false, fmt.Errorf("could not write temporary epub file: %w", err)
	}
	if err = srcFile.Close(); err != nil {
		return false, fmt.Errorf("could not close existing epub file: %w", err)
	}
	if err = workFile.Close(); err != nil {
		return false, fmt.Errorf("could not close temporary epub file: %w", err)
	}
	stdout, err := f.run(ctx, "--json-meta", "--update-epub", workFile.Name())
	if err != nil {
		return false, fmt.Errorf("could not update book: %w", err)
	}

	stdout = strings.ReplaceAll(stdout, "\r", "")
	message, rawJSON, ok := strings.Cut(stdout, "\n{\n")
	if !ok {
		f.logger.Errorf("%s", stdout)
		return false, fmt.Errorf("could not read JSON output when updating %s", book.FilePath())
	}
	f.logger.Infof("%s", message)

	doingUpdate := util.Any(strings.Split(message, "\n"), func(line string) bool {
		return strings.HasPrefix(line, "Do update -")
	})
	if !doingUpdate {
		// Update was skipped
		return false, nil
	}
	var meta meta
	if err = json.Unmarshal([]byte("{"+rawJSON), &meta); err != nil {
		f.logger.Debug("{\n" + rawJSON)
		return false, fmt.Errorf("could not read output metadata: %w", err)
	}

	updateMeta := calibre.UpdateMeta{
		Authors:   []string{meta.Author},
		Comments:  meta.Description,
		Published: meta.Published.Time,
		Publisher: meta.Publisher,
		Series:    meta.Series,
		Timestamp: meta.Updated.Time,
	}
	if err = f.calibre.UpdateBook(ctx, book.Id, updateMeta); err != nil {
		return false, fmt.Errorf("could not update book: %w", err)
	}

	return true, nil
}
