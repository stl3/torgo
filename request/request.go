package request

import (
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"

// Request is a base function for sending HTTP requests.
func Request(client *http.Client, method string, url string, header http.Header) (*http.Client, *http.Response, http.Header, error) {
	if client == nil {
		// Make a new http client with a cookie jar if no existing client is provided
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Timeout: 30 * time.Second, Jar: jar}
	}

	// Build a new request
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	// Set headers
	if header != nil {
		if _, ok := header["User-Agent"]; !ok {
			header.Set("User-Agent", defaultUserAgent)
		}
		req.Header = header
	}

	// Do request
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	return client, res, req.Header, nil
}

// Get wraps the Request function, sends an HTTP GET request, and returns the same client and the HTML of the content body.
func Get(client *http.Client, url string, headers map[string]string) (*http.Client, string, error) {
	header := http.Header{}
	for k, v := range headers {
		header.Set(k, v)
	}
	client, res, _, err := Request(client, "GET", url, header)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, "", errors.New("request failed with status: " + res.Status)
	}

	content, err := io.ReadAll(res.Body)
	return client, string(content), err
}
