// SPDX-License-Identifier: GPL-2.0-only OR GPL-3.0-only
// Copyright (C) Felix Geyer <debfx@fobos.de>

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func requestRenderFeed(response http.ResponseWriter, request *http.Request, feedRenderer *FeedRenderer) {
	url := request.URL.Query().Get("url")
	//nolint:errcheck
	response.Write(feedRenderer.renderHttpRequest(url))
}

func listenAndServe(addr string, handler http.Handler) error {
	inShutdown := false

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	go func() {
		err := srv.Serve(ln)
		if !inShutdown {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		inShutdown = true
		cancel()
	}()

	<-ctx.Done()
	return srv.Close()
}

func main() {
	feedRenderer := NewFeedRenderer()

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Heartbeat("/health"))
	router.Use(middleware.SetHeader("content-security-policy", "default-src 'self'; img-src *; form-action 'self';"))
	router.Use(middleware.SetHeader("x-frame-options", "SAMEORIGIN"))
	router.Use(middleware.Recoverer)

	router.Get("/", func(response http.ResponseWriter, request *http.Request) {
		requestRenderFeed(response, request, feedRenderer)
	})

	fileServer := http.FileServer(http.Dir("static"))
	router.Handle("/static/*", http.StripPrefix("/static", fileServer))

	err := listenAndServe(":8000", router)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
