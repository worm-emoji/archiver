package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	_ "embed"

	"github.com/worm-emoji/archiver/api"
)

var (
	apiURL = "http://localhost:8080"
	//go:embed pb.json
	pbjson []byte
)

type PinboardBookmark struct {
	Href        string    `json:"href"`
	Description any       `json:"description"`
	Extended    string    `json:"extended"`
	Meta        string    `json:"meta"`
	Hash        string    `json:"hash"`
	Time        time.Time `json:"time"`
	Shared      string    `json:"shared"`
	Toread      string    `json:"toread"`
	Tags        string    `json:"tags"`
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if os.Getenv("API_URL") != "" {
		apiURL = os.Getenv("API_URL")
	}

	var pb []PinboardBookmark
	err := json.Unmarshal(pbjson, &pb)
	check(err)

	normalized := make([]api.Bookmark, len(pb))

	for i, b := range pb {
		// some pinboard descriptions are "false" which is not a valid json string
		var title string

		d, ok := b.Description.(string)
		if ok {
			title = d
		}

		tags := strings.Split(b.Tags, " ")

		if len(tags) == 0 || tags[0] == "" {
			tags = nil
		}

		normalized[i] = api.Bookmark{
			URL:         b.Href,
			Title:       title,
			Description: b.Extended,
			Tags:        tags,
			Time:        b.Time,
		}
	}

	// make http request to api
	apiEndpoint := apiURL + "/api/add"

	body := api.AddBookmarkRequest{
		Bookmarks: normalized,
	}

	b, err := json.Marshal(body)
	check(err)

	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(b))
	check(err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("ARCHIVER_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
}
