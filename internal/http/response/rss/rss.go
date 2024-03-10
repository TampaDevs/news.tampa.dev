// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rss // import "miniflux.app/v2/internal/http/response/rss"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response"
)

// OK writes a standard RSS response with a status 200 OK.
func OK(w http.ResponseWriter, r *http.Request, body interface{}) {
	builder := response.New(w, r)
	builder.WithHeader("Content-Type", "application/rss+xml; charset=utf-8")
	builder.WithBody(body)
	builder.Write()
}

// Attachment forces the RSS document to be downloaded by the web browser.
func Attachment(w http.ResponseWriter, r *http.Request, filename string, body interface{}) {
	builder := response.New(w, r)
	builder.WithHeader("Content-Type", "application/rss+xml; charset=utf-8")
	builder.WithAttachment(filename)
	builder.WithBody(body)
	builder.Write()
}
