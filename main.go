// Provides a service for making images the size you want them to be.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	"github.com/pebble-dev/rebble-image-resizer/service"
)

type config struct {
	maxSize image.Point
	baseURL string
	listen  string
}

func parseFlags() (config, error) {
	c := config{}
	var maxSize string

	flag.StringVar(&c.baseURL, "base-url", "", "The base URL to which keys are appended")
	flag.StringVar(&c.listen, "listen", "0.0.0.0:8080", "The address to listen for connections")
	flag.StringVar(&maxSize, "max-size", "1000x1000", "The max size of an image")
	flag.Parse()

	if c.baseURL == "" {
		return c, errors.New("a base URL must be provided")
	}

	if c.listen == "" {
		return c, errors.New("a listen address is required")
	}

	r := regexp.MustCompile(`(\d+)x(\d+)`).FindStringSubmatch(maxSize)
	if r == nil {
		return c, fmt.Errorf("expected size in the format WxH, got %q", maxSize)
	}
	w, _ := strconv.Atoi(r[1])
	h, _ := strconv.Atoi(r[2])
	c.maxSize = image.Pt(w, h)
	return c, nil
}

func main() {
	c, err := parseFlags()
	if err != nil {
		log.Fatalf("Failed to parse flags: %v.\n", err)
	}
	if os.Getenv("HONEYCOMB_KEY") != "" {
		beeline.Init(beeline.Config{
			WriteKey:    os.Getenv("HONEYCOMB_KEY"),
			ServiceName: "image-resizer",
			Dataset:     "rws",
		})
	}
	r := service.New()
	r.BaseURL = c.baseURL
	r.MaxSize = c.maxSize
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("image-resizer"))
	})
	http.Handle("/", hnynethttp.WrapHandler(r))
	if err := http.ListenAndServe(c.listen, nil); err != nil {
		log.Fatalf("Serving error: %v.\n", err)
	}
}
