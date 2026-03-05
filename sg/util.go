package sg

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"

	"github.com/etnz/portfolio"
)

// diskCache implements a simple disk cache for HTTP responses
type diskCache struct {
	base   http.RoundTripper
	period portfolio.Period // nil is daily
}

// RoundTrip implements the http.RoundTripper interface. It checks for a cached
// response on disk first. If a fresh cached response is not found, it proceeds
// with the actual HTTP request and caches the new response if it's successful.
func (c *diskCache) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	// get from disk
	// diskcache implements a unique key per day, so the local tmp expires every day.
	rangeID := c.period.Range(portfolio.Today()).Identifier()
	key := fmt.Sprintf("%s %s %s", rangeID, req.Method, req.URL.String())
	key = fmt.Sprintf("%s-%x", c.period, sha1.Sum([]byte(key)))
	//key = url.PathEscape(key)

	cachedResp, err := c.get(key, req)
	if err == nil { // Cache hit
		return cachedResp, nil
	}

	resp, err = c.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	log.Printf("%v %v/%v %v", resp.Request.Method, resp.Request.URL.Host, resp.Request.URL.Path, resp.Status)
	if resp.StatusCode >= 300 {
		return resp, nil
	}
	// otherwise attempt to store it in cache

	err = c.put(key, resp)
	if err != nil {
		log.Printf("cache write err (ignored): %v\n", err)
	}
	return resp, nil
}

// get retrieves a cached response from disk
func (c *diskCache) get(key string, req *http.Request) (resp *http.Response, err error) {
	file := filepath.Join(os.TempDir(), key)
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(bytes.NewBuffer(content)), req)
}

// put stores a response to disk cache
func (c *diskCache) put(key string, resp *http.Response) (err error) {
	file := filepath.Join(os.TempDir(), key)

	content, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}

	_, err = f.Write(content)
	f.Close()
	return err
}

// newDailyCachingClient returns an http.Client that uses a disk cache where entries expire daily.
func newDailyCachingClient() *http.Client {
	client := new(http.Client)
	client.Transport = &diskCache{base: http.DefaultTransport}
	return client
}

// jwget performs an HTTP GET request to the given address and unmarshals the
// JSON response body into the provided data structure. It uses the provided
// http.Client for the request.
func jwget(client *http.Client, addr string, data interface{}) error {
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("cannot http GET %v/%v: %v", resp.Request.URL.Host, resp.Request.URL.Path, resp.Status)
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return json.Unmarshal(buf.Bytes(), data)
}
