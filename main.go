package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// SocialPresence holds the URLs for a brand's official social media pages.
type SocialPresence struct {
	FacebookURL  string `json:"facebook_url,omitempty"`
	XURL         string `json:"x_url,omitempty"` // For X / Twitter
	InstagramURL string `json:"instagram_url,omitempty"`
}

// ScrapedData holds all the information scraped from a website.
type ScrapedData struct {
	URL             string            `json:"url"`
	Timestamp       time.Time         `json:"timestamp"`
	SocialLinks     SocialPresence    `json:"social_links"`
	SecurityHeaders map[string]string `json:"security_headers"`
	Metadata        map[string]string `json:"metadata"`
	Error           string            `json:"error,omitempty"`
}

// ScrapeRequest represents the request body for scraping.
type ScrapeRequest struct {
	URLs       []string `json:"urls"`
	MaxWorkers int      `json:"max_workers"`
	RateLimit  int      `json:"rate_limit"` // in seconds
	Timeout    int      `json:"timeout"`    // in seconds
}

// ScraperConfig holds the scraper's configuration.
type ScraperConfig struct {
	URLs       []string
	MaxWorkers int
	RateLimit  time.Duration
	Timeout    time.Duration
}

// Scraper manages the scraping process.
type Scraper struct {
	config      *ScraperConfig
	collector   *colly.Collector
	logger      *log.Logger
	results     chan ScrapedData
	wg          sync.WaitGroup
	rateLimiter *time.Ticker
}

// NewScraper initializes a new Scraper instance.
func NewScraper(config *ScraperConfig) *Scraper {
	logger := log.New(os.Stdout, "scraper: ", log.LstdFlags|log.Lshortfile)
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(1),
		colly.UserAgent("NewsScraper/1.0"),
	)
	c.WithTransport(&http.Transport{
		ResponseHeaderTimeout: config.Timeout,
		TLSHandshakeTimeout:   config.Timeout,
	})
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: config.MaxWorkers,
		Delay:       config.RateLimit,
		RandomDelay: config.RateLimit / 2,
	})
	return &Scraper{
		config:      config,
		collector:   c,
		logger:      logger,
		results:     make(chan ScrapedData, len(config.URLs)),
		rateLimiter: time.NewTicker(config.RateLimit),
	}
}

// Scrape performs the scraping of a single URL.
func (s *Scraper) Scrape(url string) {
	defer s.wg.Done()
	data := ScrapedData{
		URL:             url,
		Timestamp:       time.Now(),
		SocialLinks:     SocialPresence{},
		SecurityHeaders: make(map[string]string),
		Metadata:        make(map[string]string),
	}
	c := s.collector.Clone()

	c.OnResponse(func(r *colly.Response) {
		headers := []string{"Content-Security-Policy", "Strict-Transport-Security", "X-Frame-Options", "X-Content-Type-Options"}
		for _, header := range headers {
			if value := r.Headers.Get(header); value != "" {
				data.SecurityHeaders[header] = value
			}
		}
	})

	// Scrape metadata from <meta> tags.
	c.OnHTML("meta", func(e *colly.HTMLElement) {
		name := e.Attr("name")
		content := e.Attr("content")
		if name != "" && content != "" {
			data.Metadata[name] = content
		}
	})

	// Scrape social media links from <a> tags
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		lowerLink := strings.ToLower(link)

		// Check for Facebook profile link
		if strings.Contains(lowerLink, "facebook.com/") && data.SocialLinks.FacebookURL == "" {
			if !strings.Contains(lowerLink, "sharer") && !strings.Contains(lowerLink, "plugins") {
				data.SocialLinks.FacebookURL = link
				s.logger.Printf("Found Facebook link on %s: %s", url, link)
			}
		}

		// Check for X (Twitter) profile link
		if (strings.Contains(lowerLink, "twitter.com/") || strings.Contains(lowerLink, "x.com/")) && data.SocialLinks.XURL == "" {
			if !strings.Contains(lowerLink, "intent/tweet") {
				data.SocialLinks.XURL = link
				s.logger.Printf("Found X/Twitter link on %s: %s", url, link)
			}
		}

		// Check for Instagram profile link
		if strings.Contains(lowerLink, "instagram.com/") && data.SocialLinks.InstagramURL == "" {
			data.SocialLinks.InstagramURL = link
			s.logger.Printf("Found Instagram link on %s: %s", url, link)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		data.Error = fmt.Sprintf("Failed to scrape %s: %v", url, err)
		s.logger.Printf("Error scraping %s: %v", url, err)
	})

	<-s.rateLimiter.C
	s.logger.Printf("Scraping %s", url)
	if err := c.Visit(url); err != nil {
		data.Error = fmt.Sprintf("Failed to visit %s: %v", url, err)
		s.logger.Printf("Error visiting %s: %v", url, err)
	}

	c.Wait()
	s.results <- data
}

// Run starts the concurrent scraping process.
func (s *Scraper) Run() []ScrapedData {
	s.logger.Printf("Starting scrape for %d URLs with %d workers.", len(s.config.URLs), s.config.MaxWorkers)
	for _, url := range s.config.URLs {
		s.wg.Add(1)
		go s.Scrape(url)
	}
	s.wg.Wait()
	close(s.results)
	s.rateLimiter.Stop()

	results := make([]ScrapedData, 0, len(s.config.URLs))
	for data := range s.results {
		results = append(results, data)
	}
	s.logger.Println("Scraping finished.")
	return results
}

// Global scraper lock to prevent concurrent scrapes
var scraperMutex sync.Mutex

// handleScrape handles the /api/scrape endpoint
func handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default values
	if req.MaxWorkers <= 0 {
		req.MaxWorkers = 2
	}
	if req.RateLimit <= 0 {
		req.RateLimit = 2
	}
	if req.Timeout <= 0 {
		req.Timeout = 10
	}

	if len(req.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	// Prevent concurrent scrapes
	scraperMutex.Lock()
	defer scraperMutex.Unlock()

	config := &ScraperConfig{
		URLs:       req.URLs,
		MaxWorkers: req.MaxWorkers,
		RateLimit:  time.Duration(req.RateLimit) * time.Second,
		Timeout:    time.Duration(req.Timeout) * time.Second,
	}

	scraper := NewScraper(config)
	results := scraper.Run()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleHealth handles the /api/health endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// serveFrontend serves the React frontend static files
func serveFrontend(fs http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file
		path := r.URL.Path
		if path == "/" || path == "" {
			path = "/index.html"
		}

		file, err := fs.Open(path)
		if err != nil {
			// If file not found, serve index.html for SPA routing
			path = "/index.html"
			file, err = fs.Open(path)
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
		}
		defer file.Close()

		// Determine content type
		contentType := "text/html; charset=utf-8"
		if strings.HasSuffix(path, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(path, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(path, ".svg") {
			contentType = "image/svg+xml"
		} else if strings.HasSuffix(path, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(path, ".json") {
			contentType = "application/json"
		}
		w.Header().Set("Content-Type", contentType)

		http.ServeContent(w, r, path, time.Time{}, file.(ioReadSeeker))
	}
}

type ioReadSeeker interface {
	Read([]byte) (int, error)
	Seek(int64, int) (int64, error)
}

func main() {
	// Build frontend first
	log.Println("Building frontend...")

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/scrape", handleScrape)
	mux.HandleFunc("/api/health", handleHealth)

	// Serve frontend static files from frontend/dist
	distDir := "./frontend/dist"
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		log.Printf("Warning: frontend/dist directory not found. API server will start but frontend won't be served.")
		log.Printf("Run 'npm run build' in the frontend directory first.")
	}

	// Serve static files
	mux.HandleFunc("/", serveFrontend(http.Dir(distDir)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/scrape - Scrape URLs")
	log.Printf("  GET  /api/health - Health check")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
