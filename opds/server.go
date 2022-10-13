package opds

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"golang.org/x/image/draw"
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
	mux.HandleFunc("/get/cover/", server.HandleCover)
	mux.HandleFunc("/get/thumb/", server.HandleThumb)

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

	w.Header().Set("Content-Type", "application/epub+zip")
	_, _ = io.Copy(w, file) // Ignore any errors
}

// HandleCover handles requests for /get/cover/:id
func (s *Server) HandleCover(w http.ResponseWriter, req *http.Request) {
	pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "get" || pathParts[1] != "cover" {
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

	file, err := os.Open(book.Cover)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Missing cover for book id %d", id))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Could not read cover for book id %d", id))
		}
		return
	}
	defer file.Close()

	if contentType := mime.TypeByExtension(path.Ext(book.Cover)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	_, _ = io.Copy(w, file) // Ignore any errors
}

// HandleThumb handles requests for /get/thumb/:id
func (s *Server) HandleThumb(w http.ResponseWriter, req *http.Request) {
	pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "get" || pathParts[1] != "thumb" {
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

	file, err := os.Open(book.Cover)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Missing cover for book id %d", id))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Could not read cover for book id %d", id))
		}
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to decode cover image")
		return
	}

	aspect := float64(img.Bounds().Dx()*80) / float64(img.Bounds().Dy()*60)
	width := 60
	height := 80
	if aspect > 1 {
		// Picture is squat, scale down the height
		height = int(float64(height) / aspect)
	} else {
		// Picture is tall, scale down the width
		width = int(float64(width) * aspect)
	}

	thumb := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, img.Bounds(), draw.Src, nil)

	w.Header().Set("Content-Type", "image/jpeg")
	_ = jpeg.Encode(w, thumb, nil) // Ignore any errors
}
