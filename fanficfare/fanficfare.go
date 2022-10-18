package fanficfare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/mook/fanficupdates/calibre"
	"github.com/mook/fanficupdates/model"
	"github.com/sirupsen/logrus"
)

type meta struct {
	Author      string
	FormatName  string `json:"formatname"`
	Description string `json:"description"`
	LastUpdate  string `json:"lastupdate"`
	NumChapters string `json:"numChapters"`
	Publisher   string
	Published   time.Time `json:"datePublished"`
	Series      string    `json:"series"`
	Site        string
	Status      string
	StoryURL    string `json:"storyUrl"`
	Title       string
	Updated     time.Time `json:"dateUpdated"`
	Chapters    []chapter `json:"zchapters"`
}

type chapter struct {
	Number int `json:"-"`
	Date   time.Time
	KWords string `json:"kwords"`
	Title  string
	URL    string `json:"url"`
	Words  string `json:"words"`
}

func (c *chapter) UnmarshalJSON(data []byte) error {
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
	if err = json.Unmarshal(parts[1], c); err != nil {
		return err
	}
	return nil
}

type FanFicFare struct {
	calibre        *calibre.Calibre
	supportedSites map[string]struct{}
}

func NewFanFicFare(ctx context.Context, calibre *calibre.Calibre) (*FanFicFare, error) {
	result := &FanFicFare{calibre: calibre}
	err := result.getSupportedSites(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Run FanFicFare with the given command, returning stdout.
func (f *FanFicFare) run(ctx context.Context, args ...string) (string, error) {
	args = append(args, "--run-plugin=FanFicFare", "--", "--non-interactive")
	if f.calibre.Library != "" {
		args = append(args, "--library-path="+f.calibre.Library)
	}
	cmd := exec.CommandContext(ctx, "calibre-debug", args...)
	if f.calibre.Settings != "" {
		cmd.Env = append(os.Environ(), "CALIBRE_CONFIG_DIRECTORY="+f.calibre.Settings)
	}
	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(buf), nil
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
		logrus.Infof("Skipping %s, no URL", book.Title)
		return false, nil
	}

	tld, err := publicsuffix.EffectiveTLDPlusOne(url.Hostname())
	if err != nil {
		return false, fmt.Errorf("could not get eTLD for %s: %w", url, err)
	}

	if _, ok := f.supportedSites[tld]; !ok {
		logrus.Infof("Skipping %s, not supported", url.String())
		return false, nil
	}

	logrus.Infof("Updating %s: %s", book.Title, url)
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

	message, rawJSON, ok := strings.Cut(stdout, "\n{\n")
	if !ok {
		logrus.Errorf("%s", stdout)
		return false, fmt.Errorf("could not read JSON output when updating %s", book.FilePath())
	}
	logrus.Infof("%s", message)

	if !strings.Contains(message, "Do update -") {
		// Update was skipped
		return false, nil
	}
	var meta meta
	if err = json.Unmarshal([]byte("{"+rawJSON), &meta); err != nil {
		logrus.Debugf("{\n%s", rawJSON)
		return false, fmt.Errorf("could not read output metadata: %w", err)
	}

	updateMeta := calibre.UpdateMeta{
		Authors:   []string{meta.Author},
		Comments:  meta.Description,
		Published: meta.Published,
		Publisher: meta.Publisher,
		Series:    meta.Series,
		Timestamp: meta.Updated,
	}
	if err = f.calibre.UpdateBook(ctx, book.Id, updateMeta); err != nil {
		return false, fmt.Errorf("could not update book: %w", err)
	}

	return true, nil
}
