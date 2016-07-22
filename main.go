package main

import (
    "fmt"
    "log"
    "strings"

    "./scraper"
)

func main() {
    var raw_url string
    for len(raw_url) == 0 {
        fmt.Printf("MangaHere url: ")
        fmt.Scanln(&raw_url)
    }

    if strings.ToLower(raw_url) == "exit" {
        return
    }

    series, err := scraper.NewSeries(raw_url)
    if err != nil {
        log.Fatalf("Unable to make new series: %s", err)
    }

    series.Download()

    if err = series.ZipChapters(); err != nil {
        log.Fatalf("Unable to zip chapters: %s", err)
    }

    if err = series.Cleanup(); err != nil {
        log.Fatalf("Unable to cleanup: %s", err)
    }
}
