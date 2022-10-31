// Command pdf is a chromedp example demonstrating how to capture a pdf of a
// page.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// capture pdf
	var pdf, scr []byte
	var body string
	if err := chromedp.Run(ctx, extractContent(`https://twitter.com/worm_emoji/status/1585695916547182592`, &pdf, &scr, &body)); err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("sample.pdf", pdf, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote sample.pdf")

	if err := ioutil.WriteFile("sample.png", scr, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote sample.png")
}

// print a specific pdf page.
func extractContent(urlstr string, pdf, scr *[]byte, text *string) chromedp.Tasks {
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
