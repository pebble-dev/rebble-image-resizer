package service

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

// Resizer provides an `http.Handler` that resizes the provided images.
type Resizer struct {
	BaseURL string
	MaxSize image.Point
	Mux     *mux.Router
}

// New returns an initialised `*Resizer`.
func New() *Resizer {
	r := &Resizer{}
	return r
}

func (rs *Resizer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	pathComponents := strings.Split(r.URL.Path[1:], "/")
	exactFit := false
	if len(pathComponents) == 1 {
		mimetype, content, err := fetchImageBytes(rs.BaseURL + pathComponents[0])
		if err != nil {
			http.NotFound(rw, r)
			return
		}
		addHeaders(rw.Header(), mimetype, content)
		_, _ = rw.Write(content)
		return
	}
	if pathComponents[0] == "exact" {
		exactFit = true
		if len(pathComponents) == 1 {
			http.NotFound(rw, r)
			return
		}
		pathComponents = pathComponents[1:]
	}
	// expected format: "XxY/key" or "key", so either 1 or 2 components
	if len(pathComponents) > 2 {
		http.NotFound(rw, r)
		return
	}
	maxSize := image.Point{}
	var key string
	if len(pathComponents) == 2 {
		sizeParts := strings.Split(pathComponents[0], "x")
		if len(sizeParts) != 2 {
			http.NotFound(rw, r)
			return
		}
		if sizeParts[0] == "" {
			sizeParts[0] = "0"
		}
		x, err := strconv.Atoi(sizeParts[0])
		if err != nil {
			http.NotFound(rw, r)
			return
		}
		if sizeParts[1] == "" {
			sizeParts[1] = "0"
		}
		y, err := strconv.Atoi(sizeParts[1])
		if err != nil {
			http.NotFound(rw, r)
			return
		}
		maxSize = image.Pt(x, y)
		if maxSize.X > rs.MaxSize.X {
			http.NotFound(rw, r)
			return
		}
		if maxSize.Y > rs.MaxSize.Y {
			http.NotFound(rw, r)
			return
		}
		key = pathComponents[1]
	} else {
		key = pathComponents[0]
	}

	mimetype, resized, err := rs.resizeToFit(key, maxSize, exactFit)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	addHeaders(rw.Header(), mimetype, resized)
	_, _ = rw.Write(resized)
}

func addHeaders(h http.Header, mimetype string, content []byte) {
	h.Add("Content-Type", mimetype)
	h.Add("Cache-Control", "public, max-age=2592000")
	h.Add("Content-Length", strconv.Itoa(len(content)))
}

func fetchImageBytes(url string) (string, []byte, error) {
	r, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	if r.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("couldn't fetch asset: %s (%d)", r.Status, r.StatusCode)
	}
	b, err := io.ReadAll(r.Body)
	return r.Header.Get("Content-Type"), b, err
}

func (rs *Resizer) resizeToFit(key string, size image.Point, exact bool) (string, []byte, error) {
	mimetype, b, err := fetchImageBytes(rs.BaseURL + key)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	if bytes.HasPrefix(b, []byte("GIF8")) {
		// GIFs suck. Just make sure it was the right size in the first place,
		// and assume it was reasonably well-formed.
		img, err := gif.Decode(bytes.NewBuffer(b))
		if err != nil {
			return "", nil, fmt.Errorf("failed to decode gif: %w", err)
		}
		imgSize := img.Bounds().Max
		if imgSize.Eq(size) {
			return "image/gif", b, nil
		} else {
			return "", nil, fmt.Errorf("wrong gif size: expected %dx%d but got %dx%d", imgSize.X, imgSize.Y, size.X, size.Y)
		}
	}
	img, imgFormat, err := image.Decode(bytes.NewBuffer(b))
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode image: %w", err)
	}
	if img.Bounds().Dx() == size.X && img.Bounds().Dy() == size.Y {
		return mimetype, b, nil
	}
	resized := resize.Resize(uint(size.X), uint(size.Y), img, resize.Lanczos2)
	buf := bytes.NewBuffer([]byte{})
	switch imgFormat {
	case "jpeg":
		if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: 80}); err != nil {
			return "", nil, fmt.Errorf("failed to encode jpeg: %w", err)
		}
		return "image/jpeg", buf.Bytes(), nil
	default:
		log.Printf("Unknown image type %q, default to png output", imgFormat)
		fallthrough
	case "png":
		if err := png.Encode(buf, resized); err != nil {
			return "", nil, fmt.Errorf("failed to encode png: %w", err)
		}
		return "image/png", buf.Bytes(), nil
	}
}
