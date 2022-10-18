package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"github.com/mook/fanficupdates/calibre"
	"github.com/mook/fanficupdates/fanficfare"
	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/opds"
)

type PathValue struct {
	string
}

func (p *PathValue) String() string {
	return p.string
}

func (p *PathValue) Set(input string) error {
	resolved, err := filepath.Abs(input)
	if err != nil {
		return err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", resolved)
	}
	p.string = resolved
	return nil
}

func (p *PathValue) Type() string {
	return "path"
}

func main() {
	c := &calibre.Calibre{}
	var settingsDir, libraryDir PathValue
	pflag.VarP(&settingsDir, "settings", "s", "Path to Calibre settings directory")
	pflag.VarP(&libraryDir, "library", "l", "Path to Calibre library directory")
	verbose := pflag.CountP("verbose", "v", "Produce more detailed messages")
	quiet := pflag.CountP("quiet", "q", "Produce fewer messages")
	batchSize := pflag.IntP("batch-size", "b", 0, "Update in chunks with the given chunk size")
	updateInterval := pflag.DurationP("update-interval", "i", 8*time.Hour, "Interval between successive updates")
	pflag.Parse()

	logrus.SetLevel(logrus.Level(int(logrus.InfoLevel) + *verbose - *quiet))
	if settingsDir.string != "" {
		c.Settings = settingsDir.string
	}
	if libraryDir.string != "" {
		c.Library = libraryDir.string
	}

	server := opds.NewServer()
	grp, ctx := errgroup.WithContext(context.Background())
	bookGroup := make(chan []model.CalibreBook)

	books, err := c.GetBooks(ctx)
	if err != nil {
		fmt.Printf("error getting books: %v\n", err)
		os.Exit(1)
	}
	grp.Go(func() error {
		// Batch books for updates
		if *batchSize == 0 {
			for {
				books, err := c.GetBooks(ctx)
				if err != nil {
					return fmt.Errorf("error getting books: %w", err)
				}
				bookGroup <- books
			}
		} else {
			buffer := make([]model.CalibreBook, 0, *batchSize*2)
			for {
				if len(buffer) >= *batchSize {
					bookGroup <- buffer[:*batchSize]
					buffer = buffer[*batchSize:]
					continue
				}
				books, err := c.GetBooks(ctx)
				if err != nil {
					return fmt.Errorf("error getting books: %w", err)
				}
				buffer = append(buffer, books...)
				bookGroup <- buffer[:*batchSize]
				buffer = buffer[*batchSize:]
			}
		}
	})
	grp.Go(func() error {
		// Trigger book updates
		fff, err := fanficfare.NewFanFicFare(ctx, c)
		if err != nil {
			return fmt.Errorf("error readying FanFicFare: %w", err)
		}
		for {
			func() {
				timeout, cancel := context.WithTimeout(ctx, *updateInterval)
				defer cancel()
				<-timeout.Done()
				if ctx.Err() != nil {
					// Guard against parent context closing
					return
				}
				for _, book := range <-bookGroup {
					_, err := fff.Process(ctx, book)
					if err != nil {
						logrus.Errorf("error updating %s: %v", book.Title, err)
					}
				}
			}()
		}
	})

	grp.Go(func() error {
		// Stop the server on shutdown
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		err := server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error shutting down server: %w", err)
		}
		return nil
	})
	grp.Go(func() error {
		// Start the OPDS server
		server.Books = books
		server.Addr = ":8080"
		err := server.ListenAndServe()
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("error closing server: %w", err)
	})

	if err = grp.Wait(); err != nil {
		logrus.Fatal(err)
	}
}
