package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mook/fanficupdates/calibre"
)

func main() {
	ctx := context.Background()
	c := calibre.Calibre{}
	books, err := c.GetBooks(ctx)
	if err != nil {
		fmt.Printf("error getting books: %v\n", err)
		os.Exit(1)
	}
	for _, book := range books {
		fmt.Printf("%+v\n", book)
	}
}
