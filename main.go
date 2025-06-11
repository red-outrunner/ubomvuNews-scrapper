package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/urfave/cli/v2"
)

// TrafficData holds structured information about website traffic.
type TrafficData struct {
	PageViews      string `json:"page_views,omitempty"`
	UniqueVisitors string `json:"unique_visitors,omitempty"`
	Source         string `json:"source,omitempty"` // e.g., "Meta Tag", "Inline JavaScript"
}

// ScrapedData holds all the information scraped from a website.
type ScrapedData struct {
	URL             string            `json:"url"`
	Timestamp       time.Time         `json:"timestamp"`
	Traffic         TrafficData       `json:"traffic_data"`
	SecurityHeaders map[string]string `json:"security_headers"`
	Metadata        map[string]string `json:"metadata"`
	Error           string            `json:"error,omitempty"`
}

// Config holds the scraper's configuration.
type Config struct {
	URLs       []string
	MaxWorkers int
	RateLimit  time.Duration
	OutputFile string
	Timeout    time.Duration
}

// Scraper manages the scraping process.
type Scraper struct {
	config      *Config
	collector   *colly.Collector
	logger      *log.Logger
	results     chan ScrapedData
	wg          sync.WaitGroup
	rateLimiter *time.Ticker
}

// NewScraper initializes a new Scraper instance.
func NewScraper(config *Config) *Scraper {
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
		Traffic:         TrafficData{},
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

	c.OnHTML("meta", func(e *colly.HTMLElement) {
		name := e.Attr("name")
		content := e.Attr("content")
		if name != "" && content != "" {
			data.Metadata[name] = content
			switch strings.ToLower(name) {
				case "page-views", "page_views":
					data.Traffic.PageViews = content
					data.Traffic.Source = "Meta Tag"
				case "unique-visitors", "unique_visitors":
					data.Traffic.UniqueVisitors = content
					if data.Traffic.Source == "" {
						data.Traffic.Source = "Meta Tag"
					}
			}
		}
	})

	// ** NEW: Method 1 Implementation - Search inline scripts **
	c.OnHTML("script", func(e *colly.HTMLElement) {
		// Only proceed if we haven't already found traffic data
		if data.Traffic.Source != "" {
			return
		}

		scriptContent := e.Text
		// Regex to find variable assignments containing JSON
		re := regexp.MustCompile(`(?i)(?:var|window|const|let)\s+[\w\.\_]*?(?:data|config|metrics|analytics)[\w\.\_]*?\s*=\s*({.*?});`)
		matches := re.FindStringSubmatch(scriptContent)

		if len(matches) > 1 {
			jsonString := matches[1]
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(jsonString), &jsonData); err == nil {
				// Successfully parsed JSON, now search for keys
				if pv, uv := findTrafficInMap(jsonData); pv != "" || uv != "" {
					s.logger.Printf("Found traffic data in inline script on %s", url)
					data.Traffic.PageViews = pv
					data.Traffic.UniqueVisitors = uv
					data.Traffic.Source = "Inline JavaScript"
				}
			}
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

// findTrafficInMap recursively searches a map for traffic-related keys.
func findTrafficInMap(data map[string]interface{}) (pageViews, uniqueVisitors string) {
	for key, val := range data {
		// Check top-level keys
		switch strings.ToLower(key) {
			case "pageviews", "page_views", "impressions":
				pageViews = fmt.Sprintf("%v", val)
			case "uniquevisitors", "unique_visitors", "visitors":
				uniqueVisitors = fmt.Sprintf("%v", val)
		}

		// Recursively check nested maps
		if nestedMap, ok := val.(map[string]interface{}); ok {
			pv, uv := findTrafficInMap(nestedMap)
			if pv != "" && pageViews == "" {
				pageViews = pv
			}
			if uv != "" && uniqueVisitors == "" {
				uniqueVisitors = uv
			}
		}
	}
	return
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

// SaveResults saves the scraped data to a JSON file.
func (s *Scraper) SaveResults(data []ScrapedData) error {
	file, err := os.Create(s.config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %w", s.config.OutputFile, err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data to JSON: %w", err)
	}
	s.logger.Printf("Successfully saved %d results to %s", len(data), s.config.OutputFile)
	return nil
}

// main is the entry point of the CLI application.
func main() {
	app := &cli.App{
		Name:  "news-scraper",
		Usage: "Scrape traffic, security, and metadata from news websites",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "urls",
				Aliases: []string{"u"},
				Usage:   "Comma-separated list of URLs to scrape",
				Value:   cli.NewStringSlice("https://www.news24.com", "https://www.iol.co.za", "https://businesstech.co.za"),
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file for scraped JSON data",
				Value:   "results.json",
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Usage:   "Number of concurrent scraping workers",
				Value:   2,
			},
			&cli.DurationFlag{
				Name:    "rate-limit",
				Aliases: []string{"r"},
				Usage:   "Delay between requests per domain (e.g., 2s, 500ms)",
				Value:   2 * time.Second,
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Usage:   "HTTP request timeout",
				Value:   10 * time.Second,
			},
		},
		Action: func(c *cli.Context) error {
			config := &Config{
				URLs:       c.StringSlice("urls"),
				MaxWorkers: c.Int("workers"),
				RateLimit:  c.Duration("rate-limit"),
				OutputFile: c.String("output"),
				Timeout:    c.Duration("timeout"),
			}
			scraper := NewScraper(config)
			results := scraper.Run()
			if err := scraper.SaveResults(results); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
