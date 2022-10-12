package opds

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
)

type Server struct {
	*http.Server
	Library string // Path to the library
	Books   []model.CalibreBook
}

func NewServer() *Server {
	mux := http.NewServeMux()
	server := &Server{
		Server: &http.Server{Handler: mux},
	}
	mux.HandleFunc("/opds", server.HandleCatalog)
	mux.HandleFunc("/get/epub/", server.HandleDownload)
	//mux.HandleFunc("/get/cover/", nil)
	//mux.HandleFunc("/get/thumb/", nil)

	return server
}

func writeError(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	if _, err := io.WriteString(w, msg); err != nil {
		log.Printf("Error writing error response %s: %v", msg, err)
	}
}

// HandleCatalog handles requests for path /opds
func (s *Server) HandleCatalog(w http.ResponseWriter, req *http.Request) {
	buf, err := xml.Marshal(MakeCatalog(s.Books, ""))
	if err != nil {
		log.Printf("Failed to marshal catalog: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Error rendering catalog: %v", err))
		return
	}
	_, _ = w.Write(buf)
}

// HandleDownload handles requests for path /get/epub/:id
func (s *Server) HandleDownload(w http.ResponseWriter, req *http.Request) {
	pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "get" || pathParts[1] != "epub" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid path %s", req.URL.Path))
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to convert %s to book id", pathParts[2]))
		return
	}
	book := util.Find(s.Books, func(book model.CalibreBook) bool { return book.Id == id })
	if book == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Could not find book with id %d", id))
		return
	}
	format := util.Find(book.Formats, func(f string) bool { return path.Ext(f) == ".epub" })
	if format == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Could not find epub for book id %d", id))
		return
	}

	file, err := os.Open(*format)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Missing epub for book id %d", id))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Could not read epub for book id %d", id))
		}
		return
	}
	defer file.Close()

	_, _ = io.Copy(w, file) // Ignore any errors
}
