// Command pdf is a chromedp example demonstrating how to capture a pdf of a
// page.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func main() {
	// create context
	t := time.Now()
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	fmt.Println("time to create context", time.Since(t))

	// capture pdf
	var pdf, scr []byte
	t = time.Now()
	if err := chromedp.Run(ctx, printToPDF(`https://ylukem.com`, &pdf, &scr)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("time to run", time.Since(t))

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
func printToPDF(urlstr string, pdf, scr *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.FullScreenshot(scr, 90),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().WithPrintBackground(false).Do(ctx)
			if err != nil {
				return err
			}
			*pdf = buf
			return nil
		}),
	}
}
