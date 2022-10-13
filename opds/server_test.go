package opds

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/mook/fanficupdates/model"
	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalog(t *testing.T) {
	subject := NewServer()
	server := httptest.NewServer(subject.Handler)
	defer server.Close()

	res, err := http.Get(fmt.Sprintf("%s/opds", server.URL))
	require.NoError(t, err)
	rawActual, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	prettyActual, err := util.PrettyXML(rawActual)
	require.NoError(t, err, "error pretty-printing actual output")

	// Actually testing that the catalog is valid is done in catalog_test.go
	// This just checks that the feed looks reasonable.
	assert.Contains(t, string(prettyActual), "<feed")
}

func TestDownload(t *testing.T) {
	subject := NewServer()
	subject.Books = util.RandomList(5, func() model.CalibreBook { return *makeBook(t) })
	subject.Books = append(subject.Books, *makeBook(t)) // At least one book
	server := httptest.NewServer(subject.Handler)
	defer server.Close()

	t.Run("invalid request", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/epub/", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Invalid path")
		}
	})
	t.Run("non-numeric id", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/epub/pika", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Failed to convert")
		}
	})
	t.Run("missing id", func(t *testing.T) {
		var id int
		for id = 0; id <= len(subject.Books); id++ {
			hasId := util.Any(subject.Books, func(book model.CalibreBook) bool {
				return book.Id == id
			})
			if !hasId {
				break
			}
		}

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Could not find book")
		}
	})

	t.Run("missing formats", func(t *testing.T) {
		book := &subject.Books[0]
		formats := book.Formats
		book.Formats = []string{"something.arj"}
		defer func() { book.Formats = formats }()

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Could not find epub")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		book := &subject.Books[0]
		formats := book.Formats
		defer func() { book.Formats = formats }()
		workdir := t.TempDir()
		workpath := path.Join(workdir, "hello.epub")
		book.Formats = []string{workpath}

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Missing epub")
		}
	})

	t.Run("valid format", func(t *testing.T) {
		book := &subject.Books[0]
		formats := book.Formats
		defer func() { book.Formats = formats }()
		workdir := t.TempDir()
		workpath := path.Join(workdir, "hello.epub")
		book.Formats = []string{workpath}
		expected := "pikachu"
		require.NoError(t, os.WriteFile(workpath, []byte(expected), 0o755))

		res, err := http.Get(fmt.Sprintf("%s/get/epub/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
	})
}

func TestCover(t *testing.T) {
	subject := NewServer()
	subject.Books = append(subject.Books, *makeBook(t))
	server := httptest.NewServer(subject.Handler)
	defer server.Close()

	t.Run("invalid request", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/cover/hello/world", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Invalid path")
		}
	})
	t.Run("non-numeric id", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/cover/pika", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Failed to convert")
		}
	})
	t.Run("missing id", func(t *testing.T) {
		var id int
		for id = 0; id <= len(subject.Books); id++ {
			hasId := util.Any(subject.Books, func(book model.CalibreBook) bool {
				return book.Id == id
			})
			if !hasId {
				break
			}
		}

		res, err := http.Get(fmt.Sprintf("%s/get/cover/%d", server.URL, id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Could not find book")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		workdir := t.TempDir()
		book := &subject.Books[0]
		book.Cover = path.Join(workdir, "cover.jpg")

		res, err := http.Get(fmt.Sprintf("%s/get/cover/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Missing cover")
		}
	})

	t.Run("valid cover", func(t *testing.T) {
		workdir := t.TempDir()
		book := &subject.Books[0]
		book.Cover = path.Join(workdir, "cover.jpg")
		expected := "pikachu"
		require.NoError(t, os.WriteFile(book.Cover, []byte(expected), 0o755))

		res, err := http.Get(fmt.Sprintf("%s/get/cover/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
	})
}

func TestThumb(t *testing.T) {
	subject := NewServer()
	subject.Books = append(subject.Books, *makeBook(t))
	server := httptest.NewServer(subject.Handler)
	defer server.Close()

	t.Run("invalid request", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/thumb/hello/world", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Invalid path")
		}
	})
	t.Run("non-numeric id", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("%s/get/thumb/pika", server.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Failed to convert")
		}
	})
	t.Run("missing id", func(t *testing.T) {
		var id int
		for id = 0; id <= len(subject.Books); id++ {
			hasId := util.Any(subject.Books, func(book model.CalibreBook) bool {
				return book.Id == id
			})
			if !hasId {
				break
			}
		}

		res, err := http.Get(fmt.Sprintf("%s/get/thumb/%d", server.URL, id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Could not find book")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		workdir := t.TempDir()
		book := &subject.Books[0]
		book.Cover = path.Join(workdir, "cover.jpg")

		res, err := http.Get(fmt.Sprintf("%s/get/thumb/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Missing cover")
		}
	})

	t.Run("invalid cover", func(t *testing.T) {
		workdir := t.TempDir()
		book := &subject.Books[0]
		book.Cover = path.Join(workdir, "cover.jpg")
		expected := "pikachu"
		require.NoError(t, os.WriteFile(book.Cover, []byte(expected), 0o755))

		res, err := http.Get(fmt.Sprintf("%s/get/thumb/%d", server.URL, book.Id))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Contains(t, string(body), "Failed to decode")
		}
	})

	t.Run("valid cover", func(t *testing.T) {
		type testCase struct {
			name      string
			inWidth   int
			inHeight  int
			outWidth  int
			outHeight int
		}
		testCases := []testCase{
			{"identity", 60, 80, 60, 80},
			{"half", 120, 160, 60, 80},
			{"letterbox", 80, 80, 60, 60},
			{"tall", 60, 120, 40, 80},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				workdir := t.TempDir()
				book := &subject.Books[0]
				book.Cover = path.Join(workdir, "cover.jpg")
				c := color.RGBA{255, 0, 0, 255}
				img := image.NewRGBA(image.Rect(0, 0, testCase.inWidth, testCase.inHeight))
				draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
				f, err := os.Create(book.Cover)
				require.NoError(t, err)
				err = png.Encode(f, img)
				f.Close()
				require.NoError(t, err)

				res, err := http.Get(fmt.Sprintf("%s/get/thumb/%d", server.URL, book.Id))
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, res.StatusCode)
				thumbImg, mime, err := image.Decode(res.Body)
				assert.NoError(t, err)
				assert.Equal(t, "jpeg", mime)
				assert.Equal(t, testCase.outWidth, thumbImg.Bounds().Dx(), "incorrect width")
				assert.Equal(t, testCase.outHeight, thumbImg.Bounds().Dy(), "incorrect height")
			})
		}
	})
}
