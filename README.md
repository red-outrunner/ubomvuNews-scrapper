
# 📰 news-scraper

Scrape metadata, traffic info, and security headers from South African news websites.

Built with [Colly](https://github.com/gocolly/colly) and [urfave/cli](https://github.com/urfave/cli), this CLI tool allows you to concurrently crawl multiple news sites and export structured JSON with key data for each target.

---

## ⚙️ Features

- ✅ Extracts page metadata from `<meta>` tags
- ✅ Pulls page views & unique visitors (if available in meta)
- ✅ Captures HTTP security headers
- ✅ Concurrent worker-based scraping with rate limiting
- ✅ Outputs clean JSON results
- ✅ Timeout and error handling built in

---

## 📦 Install

```bash
go build -o news-scraper main.go
```

---

## 🚀 Usage

```bash
go run main.go --urls https://www.businesstech.co.za,https://www.iol.co.za --output results.json --workers 2 --rate-limit 2s --timeout 10s
```

### Flags

| Flag            | Alias | Description                                  | Default                           |
|-----------------|-------|----------------------------------------------|-----------------------------------|
| `--urls`        | `-u`  | List of URLs to scrape                       | `https://www.news24.com`, etc.    |
| `--output`      | `-o`  | Output file for scraped JSON data            | `scraped_data.json`               |
| `--workers`     | `-w`  | Number of concurrent scraping workers        | `2`                               |
| `--rate-limit`  | `-r`  | Delay between requests per domain            | `2s`                              |
| `--timeout`     | `-t`  | HTTP request timeout                         | `10s`                             |

---

## 📄 Output Format

Each entry in the JSON array contains:

```json
{
  "url": "https://example.com",
  "timestamp": "2025-05-12T12:00:00Z",
  "page_views": "12345",
  "unique_visitors": "6789",
  "security_headers": {
    "Content-Security-Policy": "...",
    "X-Frame-Options": "...",
    ...
  },
  "metadata": {
    "description": "...",
    "keywords": "...",
    ...
  },
  "error": ""
}
```

---

## 🛠 Dev Notes

- Scraper uses `colly.Async(true)` and clones collectors per job for thread safety.
- Security headers checked: `CSP`, `HSTS`, `XFO`, `XCTO`
- Metadata is pulled from any `<meta name="...">` tag.

---

## 📋 TODOs

- [ ] Add CLI flag to include/exclude security headers or metadata
- [ ] Support more structured traffic data extraction
- [ ] Option to output CSV

---
