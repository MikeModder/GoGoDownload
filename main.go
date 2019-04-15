package main

import (
	"errors"
	"fmt"
	"flag"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	// URL format for GoGoAnime episodes
	urlString = "https://www5.gogoanime.tv/%s-episode-%d"

	// regex to find the anime name and remove special characters
	reAnimeName = regexp.MustCompile("https://vidstream.co/download\\?id=\\w+&typesub=[\\w-]+&title=(.*)")
	reSanitize  = regexp.MustCompile("[^a-zA-Z0-9\\s-_]+")

	// Variables for command line arguments
	startEp, endEp int
	seriesURL, quality string

	// Variables for govvv
	GitCommit, BuildDate, Version string
)

func init() {
	flag.StringVar(&seriesURL, "series", "", "GoGoAnime category page for the anime you want to download")
	flag.IntVar(&startEp, "start", 0, "First episode to download")
	flag.IntVar(&endEp, "end", 0, "Last episode to download")
	flag.StringVar(&quality, "quality", "720p", "Quality of video to download (one of 480p, 720p, 1080p)")

	flag.Parse()

	if startEp == 0 {
		fmt.Println("[error] you must specify a starting episode with --start!")
		os.Exit(1)
	}

	if endEp == 0 {
		fmt.Println("[error] you must specify a ending episode with --end!")
		os.Exit(1)
	}

	if seriesURL == "" {
		fmt.Println("[error] you must specify a series page with --series!")
		os.Exit(1)
	}
}

func main() {
	fmt.Printf("\tGoGoDownload\nVersion %s (%s) built %s\n", Version, GitCommit, BuildDate)

	seriesURL := os.Args[1]

	startEp, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Printf("[error] %s is not a valid number!\n", os.Args[2])
		os.Exit(1)
	}

	endEp, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Printf("[error] %s is not a valid number!\n", os.Args[2])
		os.Exit(1)
	}

	fmt.Printf("Downloading episode %d-%d of %s\n", startEp, endEp, seriesURL)

	slug := strings.Split(seriesURL, "/")[4]

	for i := startEp; i < endEp+1; i++ {
		fmt.Printf("[info] scraping episode %d\n", i)
		gga, title, err := getRapidVideoLink(fmt.Sprintf(urlString, slug, i))
		if err != nil {
			fmt.Printf("[error] failed to parse GGA for episode %d: %v\n", i, err)
			continue
		}

		mp4, err := getMp4FromRapidVideo(gga)
		if err != nil {
			fmt.Printf("[error] failed to get mp4 for episode %d: %s\n", i, err)
		}

		fmt.Printf("[info] downloading episode %d to '%s.mp4'\n", i, cleanName(title))

		c := exec.Command("aria2c", mp4, "-x", "4", "-o", fmt.Sprintf("%s.mp4", cleanName(title)))
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		c.Stdin = os.Stdin
		c.Run()
	}
}

func getRapidVideoLink(url string) (string, string, error) {
	page, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer page.Body.Close()

	if page.StatusCode != 200 {
		return "", "", errors.New("status not 200 ok")
	}

	doc, err := goquery.NewDocumentFromReader(page.Body)
	if err != nil {
		return "", "", err
	}

	url, has := doc.Find(".rapidvideo").Children().First().Attr("data-video")
	if !has {
		return "", "", errors.New("failed to find video url")
	}

	titleRaw, has := doc.Find(".download-anime").Children().First().Attr("href")
	if !has {
		return "", "", errors.New("could not find anime title")
	}

	//fmt.Printf("%v", reAnimeName.FindStringSubmatch(titleRaw))
	title, err := neturl.QueryUnescape(reAnimeName.FindStringSubmatch(titleRaw)[1])
	if err != nil {
		return "", "", err
	}

	return url, title, nil
}

func getMp4FromRapidVideo(url string) (string, error) {
	page, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer page.Body.Close()

	if page.StatusCode != 200 {
		return "", errors.New("status not 200 ok")
	}

	doc, err := goquery.NewDocumentFromReader(page.Body)
	if err != nil {
		return "", err
	}

	url, has := doc.Find("source").First().Attr("src")
	if !has {
		return "", errors.New("failed to get URL from rapidvideo")
	}

	return url, nil
}

func cleanName(name string) string {
	return reSanitize.ReplaceAllString(name, "_")
}
