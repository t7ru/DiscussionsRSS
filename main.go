package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type config struct {
	wiki     string // subdomain only: "alter-ego" -> alter-ego.fandom.com
	limit    int
	interval time.Duration
	addr     string
	title    string // optional override
	sort     string // creation_date | trending
	forum    string // optional forum ID scopes feed to one category
}

func (c *config) sourceURL() string {
	base := fmt.Sprintf("https://%s.fandom.com/wikia.php?format=json&limit=%d&sortKey=%s",
		c.wiki, c.limit, c.sort)
	if c.forum != "" {
		return base + "&controller=DiscussionForum&method=getForum&forumId=" + c.forum
	}
	return base + "&controller=DiscussionThread&method=getThreads&responseGroup=small&viewableOnly=true"
}

func (c *config) wikiBase() string { return "https://" + c.wiki + ".fandom.com" }
func (c *config) wikiHost() string { return c.wiki + ".fandom.com" }

func (c *config) feedTitle() string {
	if c.title != "" {
		return c.title
	}
	return c.wiki + " - Latest Discussions"
}

func (c *config) feedDesc() string {
	return "Latest threads from " + c.wikiHost()
}

type cachedFeeds struct {
	atom []byte
	rss  []byte
}

var cache atomic.Pointer[cachedFeeds]

func main() {
	var cfg config
	flag.StringVar(&cfg.wiki, "wiki", "", "Fandom wiki subdomain, e.g. tds for tds.fandom.com [required]")
	flag.IntVar(&cfg.limit, "limit", 20, "Threads to fetch per poll (max ~50 before Fandom paginates)")
	flag.DurationVar(&cfg.interval, "interval", 5*time.Minute, "Poll interval")
	flag.StringVar(&cfg.addr, "addr", ":7777", "HTTP listen address")
	flag.StringVar(&cfg.title, "title", "", `Feed title override (default: "<wiki> - Latest Discussions")`)
	flag.StringVar(&cfg.sort, "sort", "creation_date", "Sort order: creation_date | trending")
	flag.StringVar(&cfg.forum, "forum", "", "Forum ID to scope feed to a single category (optional)")
	flag.Parse()

	if cfg.wiki == "" {
		log.Fatal("error: -wiki is required  (e.g. -wiki tds)")
	}
	if cfg.sort != "creation_date" && cfg.sort != "trending" {
		log.Fatal("error: -sort must be creation_date or trending")
	}

	client := new(http.Client{Timeout: 15 * time.Second})

	if err := refresh(client, &cfg); err != nil {
		log.Printf("initial fetch: %v", err)
	}

	go func() {
		ticker := time.NewTicker(cfg.interval)
		for range ticker.C {
			if err := refresh(client, &cfg); err != nil {
				log.Printf("refresh: %v", err)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /feed.atom", serveAtom)
	mux.HandleFunc("GET /feed.rss", serveRSS)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "wiki:     %s\n", cfg.wikiHost())
		fmt.Fprintf(w, "sort:     %s\n", cfg.sort)
		if cfg.forum != "" {
			fmt.Fprintf(w, "forum:    %s\n", cfg.forum)
		}
		fmt.Fprintf(w, "interval: %v\n\n", cfg.interval)
		fmt.Fprintln(w, "GET /feed.atom  →  Atom 1.0")
		fmt.Fprintln(w, "GET /feed.rss   →  RSS 2.0")
	})

	log.Printf("serving %s every %v on %s", cfg.wikiHost(), cfg.interval, cfg.addr)
	log.Fatal(http.ListenAndServe(cfg.addr, mux))
}

func refresh(client *http.Client, cfg *config) error {
	entries, err := fetchThreads(client, cfg)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	a, err := buildAtom(entries, cfg)
	if err != nil {
		return fmt.Errorf("atom: %w", err)
	}
	r, err := buildRSS(entries, cfg)
	if err != nil {
		return fmt.Errorf("rss: %w", err)
	}

	cache.Store(&cachedFeeds{atom: a, rss: r})
	log.Printf("refreshed %d threads from %s", len(entries), cfg.wikiHost())
	return nil
}

func serveAtom(w http.ResponseWriter, _ *http.Request) {
	c := cache.Load()
	if c == nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Write(c.atom)
}

func serveRSS(w http.ResponseWriter, _ *http.Request) {
	c := cache.Load()
	if c == nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write(c.rss)
}
