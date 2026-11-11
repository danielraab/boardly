package widgets

import (
	"context"
	"encoding/json"
	"fmt"

	"boardly/internal/dashboards"
	"boardly/internal/feeds"
	"boardly/internal/notes"
)

// KindFeed shows a glance list of items of the owner's feeds.
const KindFeed = "feed"

// Limits of a feed widget's config.
const (
	maxFeedWidgetFeeds = 50
	maxFeedWidgetItems = 100
)

type feedConfig struct {
	// FeedIDs are subscription IDs; empty means all of the owner's feeds.
	FeedIDs      []string `json:"feedIds"`
	Show         string   `json:"show"`
	Count        int      `json:"count"`
	ShowSummary  bool     `json:"showSummary"`
	AudioOnly    bool     `json:"audioOnly"`
	Title        *string  `json:"title"`
	Size         Size     `json:"size"`
	ShowIfPublic bool     `json:"showIfPublic"`
}

// feedData is what the owner gets.
type feedData struct {
	Items []feeds.Item `json:"items"`
	// Missing says that none of the chosen feeds exist any more.
	Missing bool `json:"missing"`
}

// publicFeed is all a visitor learns about a feed: no URL and no IDs.
type publicFeed struct {
	Title     string      `json:"title"`
	Icon      *notes.Icon `json:"icon"`
	FaviconID *string     `json:"faviconId"`
}

// publicFeedItem is what a visitor gets of an item: no flags and no listen
// record.
type publicFeedItem struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Link        *string      `json:"link"`
	Summary     *string      `json:"summary"`
	PublishedAt string       `json:"publishedAt"`
	Audio       *feeds.Audio `json:"audio"`
	Feed        publicFeed   `json:"feed"`
}

type publicFeedData struct {
	Items []publicFeedItem `json:"items"`
}

// Feed returns the feed kind. Like the media widget's sources, its feeds
// don't have to exist when the layout is saved.
func Feed(store *feeds.Store) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c feedConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if c.Show != feeds.ViewLatest && c.Show != feeds.ViewReadLater && c.Show != feeds.ViewLiked && c.Show != feeds.ViewInProgress {
				return nil, fmt.Errorf("%w: show must be latest, read_later, liked or in_progress", dashboards.ErrInvalidLayout)
			}
			if c.Count < 1 || c.Count > maxFeedWidgetItems {
				return nil, fmt.Errorf("%w: count must be 1 to %d", dashboards.ErrInvalidLayout, maxFeedWidgetItems)
			}
			if len(c.FeedIDs) > maxFeedWidgetFeeds {
				return nil, fmt.Errorf("%w: at most %d feeds", dashboards.ErrInvalidLayout, maxFeedWidgetFeeds)
			}
			seen := make(map[string]bool, len(c.FeedIDs))
			for _, id := range c.FeedIDs {
				if id == "" || len(id) > maxIDLength || seen[id] {
					return nil, fmt.Errorf("%w: invalid feed", dashboards.ErrInvalidLayout)
				}
				seen[id] = true
			}
			if c.FeedIDs == nil {
				c.FeedIDs = []string{}
			}
			var err error
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			return json.Marshal(c)
		},
		// The subscriptions a widget shows are the owner's business.
		VisitorConfig: func(raw json.RawMessage) (json.RawMessage, error) {
			var c feedConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, err
			}
			c.FeedIDs = []string{}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c feedConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			if len(c.FeedIDs) > 0 {
				exists, err := store.HasSubscription(ctx, viewer.OwnerID, c.FeedIDs)
				if err != nil {
					return nil, false, err
				}
				if !exists {
					if !viewer.IsOwner {
						return nil, false, nil
					}
					return feedData{Items: []feeds.Item{}, Missing: true}, true, nil
				}
			}
			items, _, err := store.Items(ctx, viewer.OwnerID, feeds.ItemQuery{View: c.Show, SubscriptionIDs: c.FeedIDs, AudioOnly: c.AudioOnly, Limit: c.Count})
			if err != nil {
				return nil, false, err
			}
			if viewer.IsOwner {
				return feedData{Items: items}, true, nil
			}
			public := publicFeedData{Items: make([]publicFeedItem, len(items))}
			for i, it := range items {
				p := publicFeedItem{ID: it.ID, Title: it.Title, Link: it.Link, PublishedAt: it.PublishedAt, Audio: it.Audio,
					Feed: publicFeed{Title: it.Feed.Title, Icon: it.Feed.Icon, FaviconID: it.Feed.FaviconID}}
				if c.ShowSummary {
					p.Summary = it.Summary
				}
				public.Items[i] = p
			}
			return public, true, nil
		},
	}
}
