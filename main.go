package main

import (
	"errors"
	"fmt"
	"flag"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	// URL format for GoGoAnime episodes
	urlString = "https://www5.gogoanime.tv/%s-episode-%s"

	// regex to find the anime name and remove special characters
	reAnimeName = regexp.MustCompile("https://vidstream.co/download\\?id=[\\w=]+&typesub=[\\w-]+&title=(.*)")
	reSanitize  = regexp.MustCompile("[^a-zA-Z0-9\\s-_]+")
	reEpisodeNumber = regexp.MustCompile("[\\w-]+-episode-([\\d-]+)")

	// Variables for command line arguments
	startEp, endEp, downloadThreads int
	seriesURL, quality string
	debug, dryRun bool

	// Variables for govvv
	GitCommit, BuildDate, Version string
)

func init() {
	flag.StringVar(&seriesURL, "series", "", "GoGoAnime category page for the anime you want to download")
	flag.StringVar(&quality, "quality", "720p", "Quality of video to download (one of 480p, 720p, 1080p)")
	flag.IntVar(&downloadThreads, "threads", 4, "Threads to use for aria2 download")
	flag.IntVar(&startEp, "start", -1, "First episode to download")
	flag.IntVar(&endEp, "end", -1, "Last episode to download")
	flag.BoolVar(&debug, "debug", false, "Debug mode prints all sorts of garbage to your console")
	flag.BoolVar(&dryRun, "dryrun", false, "Don't actually download anything")

	flag.Parse()

	fmt.Printf("GoGoDownload - A tool to download anime from GoGoAnime\nVersion %s (%s) built %s\n", Version, GitCommit, BuildDate)

	if seriesURL == "" {
		fmt.Println("[error] you must specify a series page with --series!")
		os.Exit(1)
	}

	if startEp == -1 {
		fmt.Println("[error] you must specify a starting episode with --start!")
		os.Exit(1)
	}

	if endEp == -1 {
		fmt.Println("[error] you must specify a ending episode with --end!")
		os.Exit(1)
	}

	if quality != "480p" && quality != "720p" && quality != "1080p" {
		fmt.Println("[error] you must specify a valid quality (one of '480p', '720p' or '1080p'")
		os.Exit(1)
	}
}

func main() {
	if dryRun {
		fmt.Println("[info] Doing a dry-run, no files will be downloaded!")
	}

	slug := strings.Split(seriesURL, "/")[4]
	id, title, err := getAnimeInfoFromCategoryPage(seriesURL)
	if err != nil {
		fmt.Printf("[error] failed to get anime id from %s: %v\n", seriesURL, err)
		os.Exit(1)
	}
	debugPrint("[id: '%s', title: '%s']\n", id, title)
	seriesTitle := cleanName(title)

	eps, err := getEpisodesForID(id)
	if err != nil {
		fmt.Printf("[error] failed to get episodes: %v\n", err)
		os.Exit(1)
	}
	debugPrint("[episodes: %v]", eps)

	err = mkdir(seriesTitle)
	if err != nil {
		fmt.Printf("[error] failed to create a directory (%s) to download into: %v\n", seriesTitle, err)
		os.Exit(1)
	}

	// Make sure startEp and endEp are within bounds
	if endEp > len(eps)-1 || startEp > len(eps)-1 {
		fmt.Printf("[error] series %s does not have %d episodes, try a lower number!\n", title, endEp)
		os.Exit(1)
	}

	fmt.Printf("[info] downloading %s ep %d-%d (%d total episodes)\n", title, startEp, endEp, len(eps)-1)

	for i := startEp; i < endEp+1; i++ {
		ep := eps[i]
		fmt.Printf("[info] scraping episode %s\n", ep)
		rv, title, err := getRapidVideoLink(fmt.Sprintf(urlString, slug, ep))
		if err != nil {
			fmt.Printf("[error] failed to parse GGA for episode %s: %v\n", ep, err)
			continue
		}
		debugPrint("[rv: '%s', title: '%s']\n", rv, title)

		mp4, err := getMp4FromRapidVideo(fmt.Sprintf("%s?q=%s", rv, quality))
		if err != nil {
			fmt.Printf("[error] failed to get mp4 for episode %s: %s\n", ep, err)
			os.Exit(1)
		}
		debugPrint("[mp4: '%s']\n", mp4)

		fmt.Printf("[info] downloading episode %s to '%s.mp4'\n", ep, cleanName(title))

		if dryRun {
			continue
		}

		c := exec.Command("aria2c", mp4, "-x", string(downloadThreads), "-o", path.Join(seriesTitle, fmt.Sprintf("%s.mp4", title)))
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

	//fmt.Printf("name: %v\n", reAnimeName.FindStringSubmatch(titleRaw))
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
		// It doesn't have it in our selected resolution, so run again with a different res (scraped from the rv page)
		url, has = doc.Find("#home_video > div:nth-of-type(2) > a:last-of-type").Attr("href")
		if !has {
			return "", errors.New("failed to get URL from rapidvideo")
		}
		debugPrint("retrying mp4 scrape with new url: %s\n", url)
		fmt.Println("[info] episode was not available in the requested quality, trying a lower one")
		
		return getMp4FromRapidVideo(url)
	}

	return url, nil
}

func getAnimeInfoFromCategoryPage(url string) (string, string, error) {
	page, err := http.Get(url)
	if err != nil {
		return  "", "", err
	}
	defer page.Body.Close()

	if page.StatusCode != 200 {
		return "", "", errors.New("status code not 200 ok")
	}

	doc, err := goquery.NewDocumentFromReader(page.Body)
	if err != nil {
		return "", "", err
	}

	id, has := doc.Find(".movie_id").Attr("value")
	if !has {
		return "", "", errors.New("failed to get movie id from gga page")
	}

	title := doc.Find(".anime_info_body_bg").Find("h1").Text()

	return id, title, nil
}

func getEpisodesForID(id string) ([]string, error) {
	url := fmt.Sprintf("https://ajax.apimovie.xyz/ajax/load-list-episode?ep_start=0&ep_end=9999&id=%s", id)
	page, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer page.Body.Close()

	if page.StatusCode != 200 {
		return nil, errors.New("status code not 200 ok")
	}

	doc, err := goquery.NewDocumentFromReader(page.Body)
	if err != nil {
		return nil, err
	}

	var urls []string
	doc.Find("li a").Each(func(i int, s *goquery.Selection) {
		u, _ := s.Attr("href")
		urls = append(urls, reEpisodeNumber.FindStringSubmatch(u)[1])
	})

	return reverse(urls), nil
}

func cleanName(name string) string {
	return reSanitize.ReplaceAllString(name, "_")
}

// Stolen from https://github.com/JoshuaDoes/miitomo-assetscraper/blob/b11c5c9b21fb78eeb1f4cceb60be2e62ff399609/main.go#L254-L262
func mkdir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

// Stolen from http://golangcookbook.com/chapters/arrays/reverse/
func reverse(in []string) []string {
	for i, j := 0, len(in)-1; i < j; i, j = i+1, j-1 {
		in[i], in[j] = in[j], in[i]
	}
	return in
}

func debugPrint(s string, a ...interface{}) {
	if debug {
		fmt.Println("[debug]", fmt.Sprintf(s, a))
	}
}