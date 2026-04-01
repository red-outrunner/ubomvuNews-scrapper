import { useState } from 'react'
import './App.css'

const DEFAULT_URLS = [
  'https://www.news24.com',
  'https://www.iol.co.za',
  'https://businesstech.co.za'
]

function App() {
  const [urls, setUrls] = useState(DEFAULT_URLS.join('\n'))
  const [maxWorkers, setMaxWorkers] = useState(2)
  const [rateLimit, setRateLimit] = useState(2)
  const [timeout, setTimeout] = useState(10)
  const [results, setResults] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const handleScrape = async () => {
    const urlList = urls.split('\n').map(u => u.trim()).filter(u => u !== '')
    
    if (urlList.length === 0) {
      setError('Please enter at least one URL')
      return
    }

    setLoading(true)
    setError(null)
    setResults(null)

    try {
      const response = await fetch('/api/scrape', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          urls: urlList,
          max_workers: maxWorkers,
          rate_limit: rateLimit,
          timeout: timeout
        })
      })

      if (!response.ok) {
        const errText = await response.text()
        throw new Error(errText || 'Scraping failed')
      }

      const data = await response.json()
      setResults(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = () => {
    if (!results) return
    
    const blob = new Blob([JSON.stringify(results, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `scraped-data-${new Date().toISOString().slice(0, 10)}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <div className="app">
      <header className="header">
        <h1>📰 News Scraper</h1>
        <p className="subtitle">Scrape metadata, social links, and security headers from news websites</p>
      </header>

      <main className="main">
        <section className="config-section">
          <h2>Configuration</h2>
          
          <div className="form-group">
            <label htmlFor="urls">URLs to Scrape (one per line)</label>
            <textarea
              id="urls"
              value={urls}
              onChange={(e) => setUrls(e.target.value)}
              placeholder="Enter URLs, one per line"
              rows={6}
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label htmlFor="workers">Max Workers</label>
              <input
                type="number"
                id="workers"
                value={maxWorkers}
                onChange={(e) => setMaxWorkers(parseInt(e.target.value) || 1)}
                min="1"
                max="10"
              />
            </div>

            <div className="form-group">
              <label htmlFor="rateLimit">Rate Limit (seconds)</label>
              <input
                type="number"
                id="rateLimit"
                value={rateLimit}
                onChange={(e) => setRateLimit(parseInt(e.target.value) || 1)}
                min="1"
                max="60"
              />
            </div>

            <div className="form-group">
              <label htmlFor="timeout">Timeout (seconds)</label>
              <input
                type="number"
                id="timeout"
                value={timeout}
                onChange={(e) => setTimeout(parseInt(e.target.value) || 5)}
                min="5"
                max="60"
              />
            </div>
          </div>

          <button 
            className="scrape-button" 
            onClick={handleScrape}
            disabled={loading}
          >
            {loading ? '⏳ Scraping...' : '🚀 Start Scraping'}
          </button>
        </section>

        {error && (
          <div className="error-banner">
            <strong>Error:</strong> {error}
          </div>
        )}

        {loading && (
          <div className="loading-banner">
            <div className="spinner"></div>
            <p>Scraping in progress... This may take a moment.</p>
          </div>
        )}

        {results && (
          <section className="results-section">
            <div className="results-header">
              <h2>Results ({results.length} URLs)</h2>
              <button className="download-button" onClick={handleDownload}>
                📥 Download JSON
              </button>
            </div>

            <div className="results-grid">
              {results.map((item, index) => (
                <div key={index} className={`result-card ${item.error ? 'error' : ''}`}>
                  <div className="result-header">
                    <h3>{new URL(item.url).hostname}</h3>
                    <span className="timestamp">
                      {new Date(item.timestamp).toLocaleString()}
                    </span>
                  </div>
                  
                  {item.error ? (
                    <div className="error-message">
                      <strong>Error:</strong> {item.error}
                    </div>
                  ) : (
                    <>
                      <div className="result-section">
                        <h4>🔗 Social Links</h4>
                        <div className="social-links">
                          {item.social_links?.facebook_url && (
                            <a href={item.social_links.facebook_url} target="_blank" rel="noopener noreferrer" className="social-link facebook">
                              Facebook
                            </a>
                          )}
                          {item.social_links?.x_url && (
                            <a href={item.social_links.x_url} target="_blank" rel="noopener noreferrer" className="social-link x">
                              X / Twitter
                            </a>
                          )}
                          {item.social_links?.instagram_url && (
                            <a href={item.social_links.instagram_url} target="_blank" rel="noopener noreferrer" className="social-link instagram">
                              Instagram
                            </a>
                          )}
                          {!item.social_links?.facebook_url && !item.social_links?.x_url && !item.social_links?.instagram_url && (
                            <span className="no-data">No social links found</span>
                          )}
                        </div>
                      </div>

                      <div className="result-section">
                        <h4>🔒 Security Headers</h4>
                        <div className="headers-list">
                          {Object.keys(item.security_headers).length > 0 ? (
                            Object.entries(item.security_headers).map(([key, value]) => (
                              <div key={key} className="header-item">
                                <span className="header-name">{key}</span>
                                <span className="header-value" title={value}>{truncate(value, 50)}</span>
                              </div>
                            ))
                          ) : (
                            <span className="no-data">No security headers found</span>
                          )}
                        </div>
                      </div>

                      <div className="result-section">
                        <h4>📋 Metadata</h4>
                        <div className="metadata-list">
                          {Object.keys(item.metadata).length > 0 ? (
                            Object.entries(item.metadata).slice(0, 5).map(([key, value]) => (
                              <div key={key} className="metadata-item">
                                <span className="metadata-name">{key}</span>
                                <span className="metadata-value" title={value}>{truncate(value, 60)}</span>
                              </div>
                            ))
                          ) : (
                            <span className="no-data">No metadata found</span>
                          )}
                          {Object.keys(item.metadata).length > 5 && (
                            <span className="more-items">+{Object.keys(item.metadata).length - 5} more</span>
                          )}
                        </div>
                      </div>
                    </>
                  )}
                </div>
              ))}
            </div>
          </section>
        )}
      </main>

      <footer className="footer">
        <p>Built with Go + React</p>
      </footer>
    </div>
  )
}

function truncate(str, len) {
  if (!str) return ''
  if (str.length <= len) return str
  return str.slice(0, len) + '...'
}

export default App
