// Command pdf is a chromedp example demonstrating how to capture a pdf of a
// page.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/worm-emoji/archiver/api"
	"golang.org/x/sync/errgroup"
)

var apiURL = "http://localhost:8080"

func init() {
	if os.Getenv("API_URL") != "" {
		apiURL = os.Getenv("API_URL")
	}
}

func main() {
	hc := &http.Client{}
	ctx := context.Background()

	for ; ; time.Sleep(time.Second) {
		crawls, err := getPendingCrawls(ctx, hc)
		if err != nil {
			println(err.Error())
			continue
		}

		eg := errgroup.Group{}

		for _, url := range crawls {
			url := url
			eg.Go(func() error {
				ctx, cancel := chromedp.NewContext(ctx)
				defer cancel()

				crawl, err := crawlURL(ctx, url)
				if err != nil {
					return err
				}

				return saveCrawl(hc, crawl)
			})
		}

		err = eg.Wait()
		if err != nil {
			println(err.Error())
		}
	}

}

func getPendingCrawls(ctx context.Context, hc *http.Client) ([]string, error) {
	hr, err := http.NewRequest("GET", apiURL+"/api/crawl/pending", nil)
	if err != nil {
		return nil, err
	}

	hr.Header.Set("Authorization", "Bearer "+os.Getenv("ARCHIVER_API_KEY"))

	resp, err := hc.Do(hr)
	if err != nil {
		return nil, err
	}

	var crawls api.PendingCrawlsResponse

	err = json.NewDecoder(resp.Body).Decode(&crawls)
	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	return crawls.URLs, nil
}

func saveCrawl(hc *http.Client, req *api.AddCrawlRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	hr, err := http.NewRequest("POST", apiURL+"/api/crawl", bytes.NewReader(b))
	if err != nil {
		return err
	}

	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", "Bearer "+os.Getenv("ARCHIVER_API_KEY"))

	log.Println(fmt.Sprintf("Saving crawl for %s", req.URL))

	_, err = hc.Do(hr)
	if err != nil {
		return err
	}
	return nil
}

func crawlURL(ctx context.Context, url string) (*api.AddCrawlRequest, error) {
	// capture pdf
	var pdf, scr []byte
	var body, title string
	if err := chromedp.Run(ctx, extractContent(url, &pdf, &scr, &body, &title)); err != nil {
		return nil, err
	}

	return &api.AddCrawlRequest{
		URL:   url,
		Title: title,
		Body:  body,
	}, nil
}

// print a specific pdf page.
func extractContent(urlstr string, pdf, scr *[]byte, text, title *string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.EmulateViewport(1920, 1080),
		network.SetExtraHTTPHeaders(network.Headers(map[string]any{
			"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36",
		})),
		chromedp.Navigate(urlstr),
		chromedp.ActionFunc(func(ctx context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		}),
		chromedp.Evaluate(`document.body.innerText;`, text),
		chromedp.Evaluate(`document.title;`, title),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(false).
				Do(ctx)
			if err != nil {
				return err
			}
			*pdf = buf
			return nil
		}),
		chromedp.FullScreenshot(scr, 90),
	}
}
