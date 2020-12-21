package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/phantomjs"
)

func main() {
	fmt.Println("phantomjs loading...")
	phantomjs.DefaultProcess = phantomjs.NewProcess(phantomjs.WithIgnoreSSLErrors(true))
	if err := phantomjs.DefaultProcess.Open(); err != nil {
		panic(err)
	}
	fmt.Println("phantomjs loaded!")
	defer phantomjs.DefaultProcess.Close()

	c := make(chan os.Signal)
	//监听指定信号 ctrl+c kill
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				fmt.Println("退出", s)
				phantomjs.DefaultProcess.Close()
			default:
				fmt.Println("other", s)
			}
		}
	}()

	concurrency, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	urlQueue := make(chan string, concurrency)

	wg := &sync.WaitGroup{}
	wg.Add(len(urls))
	for i := 0; i <= concurrency; i++ {
		go startSnapshot(urlQueue, wg)
	}
	go addTask(urlQueue, urls)

	wg.Wait()
}

func startSnapshot(urlq chan string, wg *sync.WaitGroup) {
	for {
		url := <-urlq
		getSnapshot(url, wg)
	}
}

func getSnapshot(url string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		err := recover()
		if err != nil {
			println(err)
		}
	}()
	// Start the process once.
	page, err := phantomjs.CreateWebPage()
	if err != nil {
		panic(err)
	}
	defer page.Close()

	settings, err := page.Settings()
	if err != nil {
		panic(err)
	}
	settings.ResourceTimeout = 20 * time.Second
	if err := page.SetSettings(settings); err != nil {
		panic(err)
	}

	//set request headers
	requestHeader := http.Header{
		"User-Agent": []string{"Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"},
	}

	if err := page.SetCustomHeaders(requestHeader); err != nil {
		panic(err)
	}

	// Setup the viewport and render the results view.
	if err := page.SetViewportSize(1920, 1080); err != nil {
		panic(err)
	}
	// Setup the rectangular area of the web page to be rasterized when page.render is invoked.
	rect := phantomjs.Rect{
		Width:  1920,
		Height: 1080,
	}
	if err := page.SetClipRect(rect); err != nil {
		panic(err)
	}
	// Open a URL.
	fmt.Printf("%s opening...\n", url)
	if err := page.Open(url); err != nil {
		panic(err)
	}
	replaces := []string{":", "/", "\\"}
	filename := url
	for _, r := range replaces {
		filename = strings.Replace(filename, r, "_", -1)
	}
	filename = filename + ".png"
	fmt.Printf("%s rendering...\n", url)
	if err := page.Render(filename, "png", 50); err != nil {
		panic(err)
	}
	fmt.Printf("[v]%s saved!\n", filename)
}

func addTask(urlq chan string, allurl []string) {
	for _, r := range allurl {
		urlq <- r
	}
	close(urlq)
}
