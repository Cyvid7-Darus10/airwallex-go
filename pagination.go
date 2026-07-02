package airwallex

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
)

// Page is one page of a list endpoint's results, with lazy access to the
// pages after it.
//
//	page, err := client.Beneficiaries.List(ctx, nil)
//	for page != nil {
//	    for _, b := range page.Items { ... }
//	    if !page.HasMore { break }
//	    page, err = page.Next(ctx)
//	}
//
// Or walk every item across every page with All.
type Page[T any] struct {
	// Items are the results on this page.
	Items []T
	// HasMore reports whether another page follows this one.
	HasMore bool

	fetch   func(ctx context.Context, pageNum int) (*Page[T], error)
	pageNum int
}

// Next fetches the page after this one. Check HasMore first.
func (p *Page[T]) Next(ctx context.Context) (*Page[T], error) {
	if p.fetch == nil {
		return nil, fmt.Errorf("airwallex: Next called on a Page not produced by a List call")
	}
	return p.fetch(ctx, p.pageNum+1)
}

// All returns an iterator over every item on this page and all following
// pages, fetching lazily:
//
//	for item, err := range page.All(ctx) {
//	    if err != nil { ... }
//	}
func (p *Page[T]) All(ctx context.Context) iter.Seq2[T, error] {
	return iterPages(ctx, p, nil)
}

// pageEnvelope is the uniform Airwallex list response shape. Items stay
// raw so each one can be decoded with its raw JSON preserved.
type pageEnvelope struct {
	HasMore bool              `json:"has_more"`
	Items   []json.RawMessage `json:"items"`
}

// decodeItems decodes raw list items, preserving each item's raw JSON when
// the type embeds APIResource.
func decodeItems[T any](raws []json.RawMessage) ([]T, error) {
	items := make([]T, 0, len(raws))
	for _, raw := range raws {
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("airwallex: decoding list item: %w", err)
		}
		if capturer, ok := any(&item).(rawCapturer); ok {
			capturer.captureRaw(raw)
		}
		items = append(items, item)
	}
	return items, nil
}

// listPage fetches one page of a list endpoint, wiring up lazy access to
// the following pages. params may be nil.
func listPage[T any](ctx context.Context, c *Client, path string, params any) (*Page[T], error) {
	query, err := encodeQuery(params)
	if err != nil {
		return nil, err
	}
	start := 0
	if pageNum := query.Get("page_num"); pageNum != "" {
		start, err = strconv.Atoi(pageNum)
		if err != nil {
			start = 0
		}
	}
	query.Del("page_num")

	var fetch func(ctx context.Context, pageNum int) (*Page[T], error)
	fetch = func(ctx context.Context, pageNum int) (*Page[T], error) {
		pageQuery := url.Values{}
		for key, values := range query {
			pageQuery[key] = values
		}
		pageQuery.Set("page_num", strconv.Itoa(pageNum))
		var envelope pageEnvelope
		if err := c.do(ctx, http.MethodGet, path, pageQuery, nil, &envelope); err != nil {
			return nil, err
		}
		items, err := decodeItems[T](envelope.Items)
		if err != nil {
			return nil, err
		}
		return &Page[T]{
			Items:   items,
			HasMore: envelope.HasMore,
			fetch:   fetch,
			pageNum: pageNum,
		}, nil
	}
	return fetch(ctx, start)
}

// iterPages walks every item across pages. Iteration terminates when
// HasMore is false — or, defensively, when a page reports HasMore with no
// items, which would otherwise loop forever.
func iterPages[T any](ctx context.Context, first *Page[T], firstErr error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		if firstErr != nil {
			var zero T
			yield(zero, firstErr)
			return
		}
		page := first
		for {
			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}
			if !page.HasMore || len(page.Items) == 0 {
				return
			}
			next, err := page.Next(ctx)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			page = next
		}
	}
}
