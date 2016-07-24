package main

import (
    "flag"
    "fmt"
    "log"
    "strings"

    "github.com/zorchenhimer/mangahere/scraper"
)

func main() {
    var raw_url string
    var keep_dirs bool
    var force_dl bool

    flag.BoolVar(&keep_dirs, "k", false, "Do not remove (keep) the chapter directories.")
    flag.BoolVar(&force_dl, "f", false, "Force downloading already downloaded chapters.")
    flag.Parse()

    args := flag.Args()
    if len(args) > 0 {
        raw_url = args[0]
    }

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
    series.Force = force_dl

    series.Download()

    if err = series.ZipChapters(); err != nil {
        log.Fatalf("Unable to zip chapters: %s", err)
    }

    if !keep_dirs {
        if err = series.Cleanup(); err != nil {
            log.Fatalf("Unable to cleanup: %s", err)
        }
    }
}
