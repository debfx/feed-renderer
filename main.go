package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func requestRenderFeed(response http.ResponseWriter, request *http.Request, feedRenderer *FeedRenderer) {
	url := request.URL.Query().Get("url")
	//nolint:errcheck
	response.Write(feedRenderer.renderHttpRequest(url))
}

func main() {
	feedRenderer := NewFeedRenderer()

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.SetHeader("content-security-policy", "default-src 'self'; img-src *; form-action 'self';"))
	router.Use(middleware.SetHeader("x-frame-options", "SAMEORIGIN"))
	router.Use(middleware.Recoverer)

	router.Get("/", func(response http.ResponseWriter, request *http.Request) {
		requestRenderFeed(response, request, feedRenderer)
	})

	fileServer := http.FileServer(http.Dir("static"))
	router.Handle("/static/*", http.StripPrefix("/static", fileServer))

	err := http.ListenAndServe(":8000", router)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
