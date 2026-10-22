package karakeep

import (
	"cmp"
	"context"
	"net/url"
	"slices"
	"strings"
	"time"

	"boardly/internal/favourites"
)

// Settings is Karakeep's user settings shape.
type Settings struct {
	BookmarkClickAction      string   `json:"bookmarkClickAction"`
	ArchiveDisplayBehaviour  string   `json:"archiveDisplayBehaviour"`
	Timezone                 string   `json:"timezone"`
	BackupsEnabled           bool     `json:"backupsEnabled"`
	BackupsFrequency         string   `json:"backupsFrequency"`
	BackupsRetentionDays     int      `json:"backupsRetentionDays"`
	ReaderFontSize           *int     `json:"readerFontSize"`
	ReaderLineHeight         *float64 `json:"readerLineHeight"`
	ReaderFontFamily         *string  `json:"readerFontFamily"`
	AutoTaggingEnabled       *bool    `json:"autoTaggingEnabled"`
	AutoSummarizationEnabled *bool    `json:"autoSummarizationEnabled"`
	TagStyle                 string   `json:"tagStyle"`
	CuratedTagIDs            []string `json:"curatedTagIds"`
	InferredTagLang          *string  `json:"inferredTagLang"`
}

// UserSettings is users.settings. Tapping a bookmark opens the page, since
// Boardly stores no page content to show instead.
func (s *Service) UserSettings() Settings {
	return Settings{
		BookmarkClickAction:     "open_original_link",
		ArchiveDisplayBehaviour: "show",
		Timezone:                "UTC",
		BackupsFrequency:        "weekly",
		BackupsRetentionDays:    30,
		TagStyle:                "lowercase-hyphens",
	}
}

// Count is a named number in Stats.
type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DomainCount is the number of bookmarks of a domain.
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// HourCount is the number of bookmarks created in an hour of the day.
type HourCount struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// DayCount is the number of bookmarks created on a day of the week.
type DayCount struct {
	Day   int `json:"day"`
	Count int `json:"count"`
}

// SourceCount is the number of bookmarks from a source.
type SourceCount struct {
	Source *string `json:"source"`
	Count  int     `json:"count"`
}

// Stats is Karakeep's user statistics shape.
type Stats struct {
	NumBookmarks    int `json:"numBookmarks"`
	NumFavorites    int `json:"numFavorites"`
	NumArchived     int `json:"numArchived"`
	NumTags         int `json:"numTags"`
	NumLists        int `json:"numLists"`
	NumHighlights   int `json:"numHighlights"`
	BookmarksByType struct {
		Link  int `json:"link"`
		Text  int `json:"text"`
		Asset int `json:"asset"`
	} `json:"bookmarksByType"`
	TopDomains          []DomainCount `json:"topDomains"`
	TotalAssetSize      int           `json:"totalAssetSize"`
	AssetsByType        []struct{}    `json:"assetsByType"`
	BookmarkingActivity struct {
		ThisWeek    int         `json:"thisWeek"`
		ThisMonth   int         `json:"thisMonth"`
		ThisYear    int         `json:"thisYear"`
		ByHour      []HourCount `json:"byHour"`
		ByDayOfWeek []DayCount  `json:"byDayOfWeek"`
	} `json:"bookmarkingActivity"`
	TagUsage          []Count       `json:"tagUsage"`
	BookmarksBySource []SourceCount `json:"bookmarksBySource"`
}

// topN is how many domains and tags Stats lists.
const topN = 10

// UserStats is users.stats, computed from the subtree. Times are in UTC; a
// week starts on Monday, and day 0 is Sunday, as in Karakeep.
func (s *Service) UserStats(ctx context.Context, sc favourites.Scope) (Stats, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Stats{}, err
	}
	links := t.links()
	st := Stats{TopDomains: []DomainCount{}, AssetsByType: []struct{}{}, TagUsage: []Count{}}
	st.NumBookmarks = len(links)
	st.BookmarksByType.Link = len(links)
	st.NumLists = len(t.folders())
	infos := t.tagInfos()
	st.NumTags = len(infos)

	now := s.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	activity := &st.BookmarkingActivity
	activity.ByHour = make([]HourCount, 24)
	for h := range activity.ByHour {
		activity.ByHour[h].Hour = h
	}
	activity.ByDayOfWeek = make([]DayCount, 7)
	for d := range activity.ByDayOfWeek {
		activity.ByDayOfWeek[d].Day = d
	}

	domains := map[string]int{}
	for _, l := range links {
		created := l.CreatedAt.UTC()
		if !created.Before(weekStart) {
			activity.ThisWeek++
		}
		if !created.Before(monthStart) {
			activity.ThisMonth++
		}
		if !created.Before(yearStart) {
			activity.ThisYear++
		}
		activity.ByHour[created.Hour()].Count++
		activity.ByDayOfWeek[created.Weekday()].Count++
		if u, err := url.Parse(*l.URL); err == nil && u.Hostname() != "" {
			domains[strings.ToLower(u.Hostname())]++
		}
	}
	for d, n := range domains {
		st.TopDomains = append(st.TopDomains, DomainCount{d, n})
	}
	slices.SortFunc(st.TopDomains, func(a, b DomainCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Domain, b.Domain))
	})
	st.TopDomains = st.TopDomains[:min(topN, len(st.TopDomains))]
	for _, info := range infos {
		st.TagUsage = append(st.TagUsage, Count{info.Name, info.NumBookmarks})
	}
	slices.SortFunc(st.TagUsage, func(a, b Count) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Name, b.Name))
	})
	st.TagUsage = st.TagUsage[:min(topN, len(st.TagUsage))]
	st.BookmarksBySource = []SourceCount{}
	if len(links) > 0 {
		st.BookmarksBySource = append(st.BookmarksBySource, SourceCount{nil, len(links)})
	}
	return st, nil
}
