package scraper

import (
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"
)

type Series struct {
    Name        string
    Url         string  // base url? link of chapters?
    Chapters    []*Chapter
    Directory   string
    baseDir     string
}

type DLJob struct {
    Url         string
    Destination string
}

func NewSeries(url string) (*Series, error) {
    // validate url
    u := strings.Split(strings.Trim(url, "/"), "/")
    if len(u) < 5 || u[2] != "www.mangahere.co" {
        return nil, fmt.Errorf("Invalid mangahere url: %q", u[2])
    }

    if u[3] != "manga" {
        return nil, fmt.Errorf("Not a manga url: %q", u[3])
    }

    if len(u) > 5 {
        fmt.Println("Probably a chapter url.  Handle this eventually.")
    }

    fmt.Printf("Manga name: %q\n", u[4])

    // Get the manga's detail page
    base_url := strings.Join(u[0:5], "/") + "/"
    manga_det_html, err := downloadThing(base_url)
    if err != nil {
        return nil, fmt.Errorf("Unable to get manga detail page: %s", err)
    }

    s := &Series{
        Url:        base_url,
        Name:       prettifyName(u[4]),
        Directory:  u[4],
        baseDir:    "manga",
    }

    // Get the chapters
    det_urls := getUrls(manga_det_html, s.Url)
    sort.Strings(det_urls)
    for _, u := range det_urls {
        if re_urlch.MatchString(u) {
            s.AddChapter(u)
        }
    }

    return s, nil
}

func (s *Series) SetBaseDir(dir string) {
    s.baseDir = dir
}

func (s Series) String() string {
    return fmt.Sprintf("<Series Name:%q Directory:%q baseDir:%q Chapters:%d Url:%q>", s.Name, s.Directory, s.baseDir, len(s.Chapters), s.Url)
}

func (s *Series) FindChapters() error {
    data, err := downloadThing(s.Url)
    if err != nil {
        return fmt.Errorf("Unable to find chapters: %s", err)
    }

    _ = data
    return nil
}

func (s *Series) getDir() string {
    if s.baseDir != "" {
        return s.baseDir + "/" + s.Directory
    }
    return s.Directory
}

func (s *Series) AddChapter(url string) error {

    var dir string
    chNum := re_urlch.FindStringSubmatch(url)

    // Use the chapter number from MangaHere.  This accounts for extra chapters
    // quite well (eg, ch017.5).  Fall back to counting chapters and adding one.
    if len(chNum) >= 2 && chNum[1] != "" {
        dir = fmt.Sprintf("%s/c%s", s.getDir(), chNum[1])
    } else {
        dir = fmt.Sprintf("%s/c_%03d", s.getDir(), len(s.Chapters) + 1)
    }

    // Add the chapters.  This does not DL them, just creates their directories.
    c, err := NewChapter(url, dir)
    if err != nil {
        return err
    }

    s.Chapters = append(s.Chapters, c)
    return nil
}

func dlChapters(queue chan *Chapter, wg *sync.WaitGroup) {
    for c := range queue {
        if err := c.GetPageUrls(); err != nil {
            fmt.Printf("Error DLing chapter: %s\n", err)
        }
        printProgressAdd(1)
        wg.Done()
    }
}

func dlPages(queue chan *dlPageReq, wg *sync.WaitGroup) {
    for p := range queue {
        if err := p.data.Download(); err != nil {
            fmt.Printf("Error DLing page: %s\n", err)
            wg.Done()
            continue
            wg.Done()
            continue
        }

        if err := p.data.Image.WriteToFile(p.directory); err != nil {
            fmt.Printf("Error saving image: %s\n", err)
        }
        printProgressAdd(1)
        wg.Done()
    }
}

var pr_count int
var pr_lock sync.Mutex
var pr_wg sync.WaitGroup

func printProgress(notdone *bool, total int, prefix string) {
    //text := fmt.Sprintf("Chapter %03d/%03d", 0, total)
    lastLen := 0
    wenotdone := true

    for *notdone || wenotdone {
        pr_lock.Lock()
        text := fmt.Sprintf("%s % 4d/%d", prefix, pr_count, total)
        pr_lock.Unlock()

        lastLen = len(text)
        fmt.Printf(text)
        
        time.Sleep(time.Second / 8)

        for i := 0; i < lastLen; i++ {
            fmt.Printf("\b")
        }
        if *notdone == false {
            wenotdone = false
        }
    }
    //fmt.Println("")
    pr_wg.Done()
}

func printProgressAdd(delta int) {
    pr_lock.Lock()
    pr_count += delta
    pr_lock.Unlock()
}

func (s *Series) Download() {
    var wg sync.WaitGroup
    ch_queue := make(chan *Chapter, 1000)

    // Spin up some goroutines to DL chapters
    for i := 0; i < 4; i++ {
        go dlChapters(ch_queue, &wg)
    }
    fmt.Println("Go routines started")

    wg.Add(1)   // Add one to this wg so we don't prematurely close the channel

    // Add chapters to DL queue
    for _, c := range s.Chapters {
        wg.Add(1)
        ch_queue <- c
    }
    fmt.Println("Finished adding chapters to dl")
    nd := true
    pr_wg.Add(1)
    go printProgress(&nd, len(s.Chapters), "Chapter")

    // Wait for chapters to finish, then cleanup.
    wg.Done()
    wg.Wait()
    close(ch_queue)
    nd = false
    pr_wg.Wait()
    pr_count = 0

    fmt.Println("Finished DLing chapters")

    pg_queue := make(chan *dlPageReq, 10000)
    for i := 0; i < 4; i++ {
        go dlPages(pg_queue, &wg)
    }

    fmt.Println("Page goroutines started.")

    wg.Add(1)

    for _, c := range s.Chapters {
        for _, p := range c.Pages {
            wg.Add(1)
            r := &dlPageReq{
                data: p,
                directory: c.Directory,
            }
            pg_queue <- r
        }
    }
    fmt.Println("Page DL queue filled.")

    nd = true
    pr_wg.Add(1)
    length := 0
    for _, c := range s.Chapters {
        for _, _ = range c.Pages {
            length++
        }
    }

    go printProgress(&nd, length, "Page")

    wg.Done()
    wg.Wait()
    close(pg_queue)
    nd = false
    pr_wg.Wait()

    fmt.Println("Finished DLing pages")
}

