package internal

import (
	"io"
	"net/http"
	"os"
)

func downloadFile(url, tempName string) (string, error) {
	req, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer req.Body.Close()

	f, _ := os.CreateTemp("", tempName)
	defer f.Close()

	buff := make([]byte, 1024*1024)
	io.CopyBuffer(f, req.Body, buff)

	return f.Name(), nil
}
