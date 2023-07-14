package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type CobaltAPI struct {
	domain *url.URL
}

func NewAPI(domain string) *CobaltAPI {
	url, _ := url.Parse(domain)
	return &CobaltAPI{domain: url}
}

func (api *CobaltAPI) RequestTiktokInfo(url *url.URL) map[string]interface{} {
	api_url, _ := api.domain.Parse("/api/json")
	client := &http.Client{}
	body := strings.NewReader(fmt.Sprintf(`{"url": "%s", "isNoTTWatermark": true}`, url.String()))

	req, _ := http.NewRequest("POST", api_url.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, _ := client.Do(req)

	buff, _ := io.ReadAll(response.Body)
	response.Body.Close()
	var responseJson map[string]interface{}
	if err := json.Unmarshal(buff, &responseJson); err != nil {
		panic(err)
	}

	return responseJson
}

// This function downloads the file and returns the filepath. The file uses
// The tempfile API. It's up to the caller to delete the file.
func (*CobaltAPI) DownloadVideo(url string) string {
	f, _ := os.CreateTemp("", "*.mp4")

	buff := make([]byte, 1024*1024)
	resp, _ := http.Get(url)
	// Now we read the entire body of the request:
	io.CopyBuffer(f, resp.Body, buff)

	resp.Body.Close()
	f.Close()
	log.Println("Wrote file to ", f.Name())
	return f.Name()
}
