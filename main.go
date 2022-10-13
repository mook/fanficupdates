package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mook/fanficupdates/calibre"
	"github.com/mook/fanficupdates/opds"
)

func main() {
	ctx := context.Background()
	c := calibre.Calibre{}
	books, err := c.GetBooks(ctx)
	if err != nil {
		fmt.Printf("error getting books: %v\n", err)
		os.Exit(1)
	}
	server := opds.NewServer()
	server.Books = books
	server.Addr = ":8080"
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("error serving: %v", err)
	}
}
