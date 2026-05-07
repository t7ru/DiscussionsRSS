package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"
)

const maxSummary = 500

type apiResponse struct {
	Embedded struct {
		Threads []apiThread `json:"threads"`
	} `json:"_embedded"`
}

type apiThread struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	RawContent string `json:"rawContent"`
	ForumName  string `json:"forumName"`
	CreatedBy  struct {
		Name string `json:"name"`
	} `json:"createdBy"`
	CreationDate struct {
		EpochSecond int64 `json:"epochSecond"`
	} `json:"creationDate"`
}

type entry struct {
	ID        string
	Title     string
	URL       string
	Author    string
	Summary   string
	Forum     string
	Published time.Time
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Link    atomLink    `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type atomEntry struct {
	ID        string       `xml:"id"`
	Title     string       `xml:"title"`
	Link      atomLink     `xml:"link"`
	Author    atomAuthor   `xml:"author"`
	Published string       `xml:"published"`
	Updated   string       `xml:"updated"`
	Summary   string       `xml:"summary"`
	Category  atomCategory `xml:"category"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author"`
	Category    string `xml:"category"`
}

func fetchThreads(client *http.Client, cfg *config) ([]entry, error) {
	resp, err := client.Get(cfg.sourceURL())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var api apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, err
	}

	base := cfg.wikiBase()
	entries := make([]entry, 0, len(api.Embedded.Threads))
	for _, t := range api.Embedded.Threads {
		entries = append(entries, entry{
			ID:        t.ID,
			Title:     t.Title,
			URL:       base + "/f/p/" + t.ID,
			Author:    t.CreatedBy.Name,
			Summary:   truncate(t.RawContent, maxSummary),
			Forum:     t.ForumName,
			Published: time.Unix(t.CreationDate.EpochSecond, 0).UTC(),
		})
	}
	return entries, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	count := 0
	for i := range s {
		if count == max {
			return s[:i] + "…"
		}
		count++
	}
	return s
}

func buildAtom(entries []entry, cfg *config) ([]byte, error) {
	updated := time.Now().UTC().Format(time.RFC3339)
	if len(entries) > 0 {
		updated = entries[0].Published.Format(time.RFC3339)
	}

	base := cfg.wikiBase()
	host := cfg.wikiHost()
	feed := atomFeed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   cfg.feedTitle(),
		ID:      base + "/f",
		Updated: updated,
		Link:    atomLink{Href: base + "/f", Rel: "alternate"},
	}

	for _, e := range entries {
		pub := e.Published.Format(time.RFC3339)
		feed.Entries = append(feed.Entries, atomEntry{
			ID:        "tag:" + host + ",2024:thread:" + e.ID,
			Title:     e.Title,
			Link:      atomLink{Href: e.URL, Rel: "alternate"},
			Author:    atomAuthor{Name: e.Author},
			Published: pub,
			Updated:   pub,
			Summary:   e.Summary,
			Category:  atomCategory{Term: e.Forum},
		})
	}

	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func buildRSS(entries []entry, cfg *config) ([]byte, error) {
	pubDate := time.Now().UTC().Format(time.RFC1123Z)
	if len(entries) > 0 {
		pubDate = entries[0].Published.Format(time.RFC1123Z)
	}

	base := cfg.wikiBase()
	ch := rssChannel{
		Title:       cfg.feedTitle(),
		Link:        base + "/f",
		Description: cfg.feedDesc(),
		PubDate:     pubDate,
	}

	for _, e := range entries {
		ch.Items = append(ch.Items, rssItem{
			Title:       e.Title,
			Link:        e.URL,
			Description: e.Summary,
			PubDate:     e.Published.Format(time.RFC1123Z),
			GUID:        e.URL,
			Author:      e.Author,
			Category:    e.Forum,
		})
	}

	out, err := xml.MarshalIndent(rssFeed{Version: "2.0", Channel: ch}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func prettyFormat(wiki string) string {
	return fmt.Sprintf("%s - Latest Discussions", wiki)
}
