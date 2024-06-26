// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/response/rss"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showPublicHomepage(w http.ResponseWriter, r *http.Request) {
	categories, err := h.store.HomepageDefaultCategories()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if categories == nil {
		html.NotFound(w, r)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)
	builder := storage.NewAnonymousQueryBuilder(h.store)
	builder.WithCategories(categories)
	builder.WithSorting("published_at", "desc")
	builder.WithOffset(offset)
	builder.WithLimit(25)

	entries, err := builder.GetEntries()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	count, err := builder.CountEntries()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("categories", categories)
	view.Set("total", count)
	view.Set("entries", entries)
	view.Set("pagination", getPagination("/", count, offset, 25))
	view.Set("menu", "categories")
	view.Set("countUnread", h.store.CountUnreadEntries(1))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(1))
	view.Set("rss_url", "https://"+r.Host+route.Path(h.router, "home_rss"))
	view.Set("hasSaveEntry", h.store.HasSaveEntry(1))
	view.Set("showOnlyUnreadEntries", false)

	html.OK(w, r, view.Render("category_entries_public"))
}

func (h *handler) publicHomepageRSS(w http.ResponseWriter, r *http.Request) {
	categories, err := h.store.HomepageDefaultCategories()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if categories == nil {
		html.NotFound(w, r)
		return
	}

	builder := storage.NewAnonymousQueryBuilder(h.store)
	builder.WithCategories(categories)
	builder.WithSorting("published_at", "desc")
	builder.WithLimit(100)

	entries, err := builder.GetEntries()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("entries", entries)
	view.Set("rss", true)
	view.Set("rss_url", "https://"+r.Host+route.Path(h.router, "home_rss"))
	view.Set("title", "Tampa Devs News - Homepage Feed")
	view.Set("description", locale.NewPrinter("en_US").Printf("home_page.rss.description"))

	rss.OK(w, r, view.Render("category_entries_public"))
}
