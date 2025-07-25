package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a token bucket algorithm
type RateLimiter struct {
	rate       int
	buckets    map[string]int
	lastUpdate map[string]time.Time
	mutex      sync.RWMutex
}

func NewRateLimiter(rate int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		buckets:    make(map[string]int),
		lastUpdate: make(map[string]time.Time),
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	if _, exists := rl.buckets[ip]; !exists {
		rl.buckets[ip] = rl.rate
		rl.lastUpdate[ip] = now
		return true
	}

	elapsed := now.Sub(rl.lastUpdate[ip])
	tokensToAdd := int(elapsed.Seconds()) * rl.rate

	if tokensToAdd > 0 {
		rl.buckets[ip] = min(rl.rate, rl.buckets[ip]+tokensToAdd)
		rl.lastUpdate[ip] = now
	}

	if rl.buckets[ip] > 0 {
		rl.buckets[ip]--
		return true
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validateAndFixURL ensures the URL has a proper scheme
func validateAndFixURL(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL format: scheme and host are required")
	}

	return rawURL, nil
}

// IPProtection tracks suspicious behavior
type IPProtection struct {
	failedRequests int
	lastRequest    time.Time
	patterns       map[string]int // Track request patterns
	blacklisted    bool
	totalBytes     int64
}

type SecurityConfig struct {
	maxBodySize      int64
	maxFailedReqs    int
	blacklistTimeout time.Duration
	patternLimit     int
}

type EnhancedProxy struct {
	rateLimiter *RateLimiter
	protection  map[string]*IPProtection
	blacklist   map[string]time.Time
	mutex       sync.RWMutex
	securityCfg SecurityConfig
}

func NewEnhancedProxy(rate int) *EnhancedProxy {
	return &EnhancedProxy{
		rateLimiter: NewRateLimiter(rate),
		protection:  make(map[string]*IPProtection),
		blacklist:   make(map[string]time.Time),
		securityCfg: SecurityConfig{
			maxBodySize:      1024 * 1024, // 1MB
			maxFailedReqs:    5,
			blacklistTimeout: 5 * time.Minute,
			patternLimit:     10,
		},
	}
}

func (ep *EnhancedProxy) isBlacklisted(ip string) bool {
	ep.mutex.RLock()
	defer ep.mutex.RUnlock()

	if banTime, exists := ep.blacklist[ip]; exists {
		if time.Since(banTime) < ep.securityCfg.blacklistTimeout {
			return true
		}
		delete(ep.blacklist, ip)
	}
	return false
}

func (ep *EnhancedProxy) checkRequest(r *http.Request, ip string) error {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	if ep.protection[ip] == nil {
		ep.protection[ip] = &IPProtection{
			patterns: make(map[string]int),
		}
	}

	prot := ep.protection[ip]

	// Check request pattern (URL + Method)
	pattern := r.Method + r.URL.Path
	prot.patterns[pattern]++

	if prot.patterns[pattern] > ep.securityCfg.patternLimit {
		ep.blacklist[ip] = time.Now()
		return fmt.Errorf("suspicious pattern detected")
	}

	// Check payload size
	if r.ContentLength > ep.securityCfg.maxBodySize {
		prot.failedRequests++
		return fmt.Errorf("request too large")
	}

	return nil
}

func main() {
	targetURL := flag.String("url", "localhost:8080", "target URL to proxy")
	limit := flag.Int("limit", 1, "requests per second limit")
	port := flag.String("port", "3000", "port to run proxy on")
	maxSize := flag.Int64("maxsize", 1024*1024, "max request size in bytes")
	flag.Parse()

	fixedURL, err := validateAndFixURL(*targetURL)
	if err != nil {
		log.Fatal("Invalid target URL:", err)
	}

	target, err := url.Parse(fixedURL)
	if err != nil {
		log.Fatal("Failed to parse target URL:", err)
	}

	proxy := NewEnhancedProxy(*limit)
	proxy.securityCfg.maxBodySize = *maxSize

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Proxy error: "+err.Error(), http.StatusBadGateway)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}

		if proxy.isBlacklisted(ip) {
			http.Error(w, "IP blacklisted", http.StatusForbidden)
			return
		}

		if err := proxy.checkRequest(r, ip); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		if !proxy.rateLimiter.Allow(ip) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		reverseProxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    ":" + *port,
		Handler: handler,
	}

	log.Printf("Starting reverse proxy server on port %s", *port)
	log.Printf("Proxying requests to: %s", fixedURL)
	log.Printf("Rate limit: %d requests per second per IP", *limit)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server error:", err)
	}
}
