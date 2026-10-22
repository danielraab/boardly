package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"boardly/internal/config"
)

const healthcheckTimeout = 3 * time.Second

// runHealthcheck checks the server on the local host at BOARDLY_PORT. It reads
// no other setting, so an unrelated configuration problem doesn't turn into a
// misleading health result.
func runHealthcheck(lookupEnv func(string) (string, bool)) error {
	port, err := config.LoadPort(lookupEnv)
	if err != nil {
		return err
	}
	return checkHealth("http://127.0.0.1:"+strconv.Itoa(port)+"/api/health", healthcheckTimeout)
}

// checkHealth requests url and returns nil only for status 200.
func checkHealth(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}
	return nil
}
