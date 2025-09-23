package amundi

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const amundiSessionFile = "pcs-amundi-session"

func LoadHeaders() (http.Header, error) {
	sessionPath := filepath.Join(os.TempDir(), amundiSessionFile)
	headerData, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("amundi session not found. Please run 'pcs amundi login' first: %w", err)
	}

	headers := make(http.Header)
	scanner := bufio.NewScanner(strings.NewReader(string(headerData)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	return headers, nil
}
