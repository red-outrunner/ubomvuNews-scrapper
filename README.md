# 📰 News Scraper

A modern web application for scraping metadata, social media links, and security headers from news websites.

Built with **Go** (backend API) and **React** (frontend).

---

## ⚙️ Features

- ✅ **Web-based UI** - Modern React interface
- ✅ Extracts page metadata from `<meta>` tags
- ✅ Captures social media links (Facebook, X/Twitter, Instagram)
- ✅ Captures HTTP security headers
- ✅ Concurrent worker-based scraping with rate limiting
- ✅ Exports clean JSON results
- ✅ Timeout and error handling built in
- ✅ **No Go 1.23+ required** - Works with Go 1.19+

---

## 📦 Installation

### Prerequisites

- Go 1.19 or later
- Node.js 18+ and npm

### Build

```bash
# Option 1: Use the start script (recommended)
chmod +x start.sh
./start.sh

# Option 2: Manual build
# Build frontend
cd frontend
npm install
npm run build
cd ..

# Build backend
go build -o news-scraper main.go

# Run
./news-scraper
```

---

## 🚀 Usage

### Start the Server

```bash
./news-scraper
```

The server will start on `http://localhost:8080`

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT`   | Server port | `8080`  |

### API Endpoints

#### POST `/api/scrape`

Scrape the provided URLs.

**Request Body:**
```json
{
  "urls": ["https://example.com", "https://news24.com"],
  "max_workers": 2,
  "rate_limit": 2,
  "timeout": 10
}
```

**Response:**
```json
[
  {
    "url": "https://example.com",
    "timestamp": "2026-02-22T21:00:00Z",
    "social_links": {
      "facebook_url": "https://facebook.com/example",
      "x_url": "https://twitter.com/example",
      "instagram_url": "https://instagram.com/example"
    },
    "security_headers": {
      "Strict-Transport-Security": "max-age=31536000",
      "X-Frame-Options": "DENY"
    },
    "metadata": {
      "description": "Example website",
      "keywords": "example, demo"
    }
  }
]
```

#### GET `/api/health`

Health check endpoint.

**Response:**
```json
{
  "status": "ok"
}
```

---

## 🏗️ Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   React UI      │────▶│   Go HTTP API    │────▶│   Target Sites  │
│   (frontend)    │     │   (main.go)      │     │   (news sites)  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

- **Frontend**: React + Vite (served as static files)
- **Backend**: Go HTTP server with Colly scraper
- **Communication**: REST API (JSON)

---

## 📄 Output Format

Each scraped result contains:

```json
{
  "url": "https://example.com",
  "timestamp": "2026-02-22T21:00:00Z",
  "social_links": {
    "facebook_url": "...",
    "x_url": "...",
    "instagram_url": "..."
  },
  "security_headers": {
    "Content-Security-Policy": "...",
    "Strict-Transport-Security": "...",
    "X-Frame-Options": "...",
    "X-Content-Type-Options": "..."
  },
  "metadata": {
    "description": "...",
    "keywords": "...",
    "author": "..."
  },
  "error": ""
}
```

---

## 🛠️ Development

### Running in Development

**Terminal 1 - Frontend (Vite dev server):**
```bash
cd frontend
npm run dev
```

**Terminal 2 - Backend (Go server):**
```bash
go run main.go
```

Then access the frontend at `http://localhost:5173` (Vite) or the Go server at `http://localhost:8080` (production build).

### Project Structure

```
ubomvuNews-scrapper/
├── main.go           # Go HTTP API server
├── go.mod            # Go module definition
├── frontend/         # React application
│   ├── src/
│   │   ├── App.jsx   # Main React component
│   │   ├── App.css   # Styles
│   │   └── main.jsx  # Entry point
│   ├── index.html
│   └── package.json
└── start.sh          # Build & start script
```

---

## 📋 Technologies

| Component | Technology |
|-----------|------------|
| Backend   | Go 1.19+, Colly |
| Frontend  | React 19, Vite |
| Styling   | CSS3 |

---

## 🔒 Security Notes

- The scraper respects `robots.txt` via Colly
- Rate limiting is built-in to avoid overwhelming target sites
- Configurable timeout prevents hanging requests

---
