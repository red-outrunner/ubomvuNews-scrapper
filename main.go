package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"github.com/gocolly/colly/v2"
	"github.com/urfave/cli/v2"
)

// ScrapedData information for a website
type ScrapedData struct {
	URL            string            `json:"url"`
	Timestamp      time.Time         `json:"timestamp"`
	PageViews      string            `json:"page_views"`
	UniqueVisitors string            `json:"unique_visitors"`
	SecurityHeaders map[string]string `json:"security_headers"`
	Metadata       map[string]string `json:"metadata"`
	Error          string            `json:"error,omitempty"`
}

// scraper configuration
type Config struct {
	URLs        []string
	MaxWorkers  int
	RateLimit   time.Duration
	OutputFile  string
	Timeout     time.Duration
}

// the scraping process
type Scraper struct {
	config     *Config
	collector  *colly.Collector
	logger     *log.Logger
	results    chan ScrapedData
	wg         sync.WaitGroup
	rateLimiter *time.Ticker
}

// NewScraper
func NewScraper(config *Config) *Scraper {
	logger := log.New(os.Stdout, "scraper: ", log.LstdFlags|log.Lshortfile)
	c := colly.NewCollector(
		colly.Async(true),
				colly.MaxDepth(1),
				colly.UserAgent("NewsScraper/1.0 (+https://example.com)"),
	)

	// request timeout
	c.WithTransport(&http.Transport{
		TLSHandshakeTimeout:   config.Timeout,
		ResponseHeaderTimeout: config.Timeout,
	})

	// Optimization of thy memory
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

func (s *Scraper) Scrape(url string) {
	defer s.wg.Done()
	data := ScrapedData{
		URL:            url,
		Timestamp:      time.Now(),
		SecurityHeaders: make(map[string]string),
		Metadata:       make(map[string]string),
	}

	// Clone collector for thread-safety
	c := s.collector.Clone()

	// Handle HTTP response
	c.OnResponse(func(r *colly.Response) {
		// Extract security headers
		for _, header := range []string{
			"Content-Security-Policy",
			"X-Frame-Options",
			"X-Content-Type-Options",
			"Strict-Transport-Security",
		} {
			if value := r.Headers.Get(header); value != "" {
				data.SecurityHeaders[header] = value
			}
		}
	})

	// Parse HTML content
	c.OnHTML("meta", func(e *colly.HTMLElement) {
		name := e.Attr("name")
		content := e.Attr("content")
		if name != "" && content != "" {
			data.Metadata[name] = content
		}
	})

	c.OnHTML("meta[name='page-views']", func(e *colly.HTMLElement) {
		data.PageViews = e.Attr("content")
	})
	c.OnHTML("meta[name='unique-visitors']", func(e *colly.HTMLElement) {
		data.UniqueVisitors = e.Attr("content")
	})

	// Error handling - very hell
	c.OnError(func(r *colly.Response, err error) {
		data.Error = fmt.Sprintf("Failed to scrape %s: %v", url, err)
		s.logger.Printf("Error scraping %s: %v", url, err)
	})

	// Rate limit
	<-s.rateLimiter.C
	s.logger.Printf("Scraping %s", url)
	if err := c.Visit(url); err != nil {
		data.Error = fmt.Sprintf("Failed to visit %s: %v", url, err)
		s.logger.Printf("Error visiting %s: %v", url, err)
	}

	c.Wait()
	s.results <- data
}

// Run starts the scraping process for all URLs listed - 6 for sure
func (s *Scraper) Run() []ScrapedData {
	for _, url := range s.config.URLs {
		s.wg.Add(1)
		go s.Scrape(url)
	}

	s.wg.Wait()
	close(s.results)

	var results []ScrapedData
	for data := range s.results {
		results = append(results, data)
	}
	return results
}

// SaveResults
func (s *Scraper) SaveResults(data []ScrapedData) error {
	file, err := os.Create(s.config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data: %v", err)
	}
	s.logger.Printf("Results saved to %s", s.config.OutputFile)
	return nil
}

func main() {
	app := &cli.App{
		Name:  "news-scraper",
		Usage: "Scrape traffic, security, and metadata from South African news websites",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "urls",
				Aliases: []string{"u"},
				Usage:   "URLs of websites to scrape",
				Value:   cli.NewStringSlice("https://www.news24.com", "https://www.iol.co.za","https://businesstech.co.za"),
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file for scraped data",
				Value:   "scraped_data.json",
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Usage:   "Number of concurrent workers",
				Value:   2,
			},
			&cli.DurationFlag{
				Name:    "rate-limit",
				Aliases: []string{"r"},
				Usage:   "Rate limit between requests",
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
				return err
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
