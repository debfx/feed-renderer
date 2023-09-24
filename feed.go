// SPDX-License-Identifier: GPL-2.0-only OR GPL-3.0-only
// Copyright (C) Felix Geyer <debfx@fobos.de>

package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

const TEMPLATE_HEADER = `
<!DOCTYPE html>
<html>
<head>
  <title>{{.title}}</title>
  <link href="/static/main.css" rel="stylesheet">
</head>
<body>
<h1>RSS/Atom feed renderer</h1>
<form action="/" method="get">
  <label for="url">URL:</label>
  <input type="url" id="url" name="url" value="{{.url}}" size="50"><br/>
  <input type="submit" id="submit" value="Render">
</form>
`

var FOOTER = []byte(`
</body>
</html>
`)

const TEMPLATE_FEED = `
<h2><a href="{{.Link}}" rel="nofollow">{{.Title}}</a></h2>

{{range .Items}}
<div class="entry">
  <h3><a href="{{.Link}}" rel="nofollow">{{.Title}}</a></h3>
  <div class="content">
    {{if .Content}} {{.Content|sanitizeHTML|safeHTML}} {{else}} {{.Description|sanitizeHTML|safeHTML}} {{end}}
  </div>
  {{if .PublishedParsed}}
  <small>
    Published: {{.PublishedParsed.Format "2006-01-02 15:04:05 -0700"}}
  </small>
  {{end}}
</div>
{{end}}

</body>
</html>
`

const TEMPLATE_ERROR = `
<h1>Error rendering the feed</h1>

<div class="alert">
  {{.error}}<br/>
  <a href="{{.url}}">{{.url}}</a>
</div>

</body>
</html>
`

type FeedRenderer struct {
	parser         *gofeed.Parser
	contentPolicy  *bluemonday.Policy
	templateHeader *template.Template
	templateFeed   *template.Template
	templateError  *template.Template
}

func NewFeedRenderer() *FeedRenderer {
	feedRenderer := &FeedRenderer{
		parser:        gofeed.NewParser(),
		contentPolicy: bluemonday.UGCPolicy(),
	}

	templateFuncMap := template.FuncMap{
		"sanitizeHTML": feedRenderer.contentPolicy.Sanitize,
		"safeHTML": func(html string) template.HTML {
			return template.HTML(html)
		},
	}

	feedRenderer.templateHeader = template.Must(
		template.New("templateHeader").Funcs(templateFuncMap).Parse(TEMPLATE_HEADER),
	)
	feedRenderer.templateFeed = template.Must(
		template.New("templateFeed").Funcs(templateFuncMap).Parse(TEMPLATE_FEED),
	)
	feedRenderer.templateError = template.Must(
		template.New("templateError").Funcs(templateFuncMap).Parse(TEMPLATE_ERROR),
	)

	return feedRenderer
}

func (fr *FeedRenderer) render(feedUrl string) (result []byte, title string, err error) {
	urlParsed, err := url.Parse(feedUrl)
	if err != nil {
		return result, title, fmt.Errorf("invalid url")
	}
	if urlParsed.Scheme != "http" && urlParsed.Scheme != "https" {
		return result, title, fmt.Errorf("invalid url protocol, only http and https are allowed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	feed, err := fr.parser.ParseURLWithContext(feedUrl, ctx)
	if err != nil {
		return result, title, err
	}

	var output bytes.Buffer
	err = fr.templateFeed.Execute(&output, feed)
	if err != nil {
		return result, title, err
	}
	result = output.Bytes()
	title = feed.Title
	return result, title, nil
}

func (fr *FeedRenderer) renderHeader(title string, url string) []byte {
	templateData := make(map[string]string)
	templateData["title"] = title
	templateData["url"] = url

	var output bytes.Buffer
	err := fr.templateHeader.Execute(&output, templateData)
	if err != nil {
		panic(err)
	}
	return output.Bytes()
}

func (fr *FeedRenderer) renderError(errorMessage string, url string) []byte {
	templateData := make(map[string]string)
	templateData["url"] = url
	templateData["error"] = errorMessage

	var output bytes.Buffer
	err := fr.templateError.Execute(&output, templateData)
	if err != nil {
		panic(err)
	}
	return output.Bytes()
}

func (fr *FeedRenderer) renderHttpRequest(url string) []byte {
	var header []byte
	var main []byte

	if url != "" {
		mainFeed, title, err := fr.render(url)
		if err != nil {
			header = fr.renderHeader("Error rendering feed", url)
			main = fr.renderError(fmt.Sprintf("%v", err), url)
		} else {
			header = fr.renderHeader(title, url)
			main = mainFeed
		}
	} else {
		header = fr.renderHeader("RSS/Atom feed renderer", url)
	}

	result := append(header, main...)
	result = append(result, FOOTER...)
	return result
}
