package main

import (
	"cmp"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// runHealthcheck probes this process's own /health/live endpoint and returns
// a process exit code, for use as `ENTRYPOINT ["/api", "-healthcheck"]` in
// environments (like distroless) that have no shell-based probe tool.
func runHealthcheck() int {
	addr := cmp.Or(os.Getenv("HTTP_ADDR"), ":8080")
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = strings.TrimPrefix(addr, ":")
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health/ready")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
