package scraper

import (
    "fmt"
    "sort"
    "strings"
    "sync"
)

type Series struct {
    Name        string
    Url         string  // manga detail page url
    Chapters    []*Chapter
    Directory   string
    baseDir     string
}

var num_dl_threads int = 4
const retry_count int = 4

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
            s.addChapter(u)
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

func (s *Series) getDir() string {
    if s.baseDir != "" {
        return s.baseDir + "/" + s.Directory
    }
    return s.Directory
}

func (s *Series) Cleanup() error {
    for _, c := range s.Chapters {
        if ex, _ := exists(c.Directory + ".zip"); ex == false {
            continue
        }
        if err := c.rmr(); err != nil {
            return err
        }
    }
    return nil
}

func (s *Series) addChapter(url string) error {

    var name string
    chNum := re_urlch.FindStringSubmatch(url)

    // Use the chapter number from MangaHere.  This accounts for extra chapters
    // quite well (eg, ch017.5).  Fall back to counting chapters and adding one.
    if len(chNum) >= 2 && chNum[1] != "" {
        name = fmt.Sprintf("c%s", chNum[1])
    } else {
        name = fmt.Sprintf("c_%03d", len(s.Chapters) + 1)
    }

    dir := fmt.Sprintf("%s/%s", s.getDir(), name)

    // Add the chapters.  This does not DL them, just creates their directories.
    c, err := NewChapter(url, dir)
    if err != nil {
        return err
    }
    c.Name = name

    s.Chapters = append(s.Chapters, c)
    return nil
}

func dlChapters(queue chan *Chapter, wg *sync.WaitGroup) {
    for c := range queue {
        if err := c.getPageUrls(); err != nil {
            if err == emptyChapter {
                if err = c.rmdir(); err != nil {
                    fmt.Printf("\nError deleting empty chapter directory: %s\n", err)
                }
            } else {
                fmt.Printf("\nError DLing chapter: %s\n", err)
            }
        }
        printProgressAdd(1)
        wg.Done()
    }
}

func dlPages(queue chan *dlPageReq, wg *sync.WaitGroup) {
    var err error
    for p := range queue {
        // Retry downloading the page retry_count times.  Give up after that
        // many uncessful attempts.
        for try := 0; try < retry_count; try++ {
            if err = p.data.download(); err == nil {
                break
            } else {
                fmt.Printf("\nError downloading page, retrying. [%d/%d]\n", try + 1, retry_count)
            }
        }
        if err != nil {
            fmt.Printf("\nToo many failed attempts to download page: %s\n", err)
            wg.Done()
            continue
        }

        if err = p.data.Image.writeToFile(p.directory); err != nil {
            fmt.Printf("\nError saving image: %s\n", err)
        }
        printProgressAdd(1)
        wg.Done()
    }
}

// TODO return errors or something from here?
func (s *Series) Download() {
    var wg sync.WaitGroup
    ch_queue := make(chan *Chapter, 1000)

    // Spin up some goroutines to DL chapters
    for i := 0; i < num_dl_threads; i++ {
        go dlChapters(ch_queue, &wg)
    }

    wg.Add(1)   // Add one to this wg so we don't prematurely close the channel

    // Add chapters to DL queue
    for _, c := range s.Chapters {
        wg.Add(1)
        ch_queue <- c
    }
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
    for i := 0; i < num_dl_threads; i++ {
        go dlPages(pg_queue, &wg)
    }

    wg.Add(1)

    for _, c := range s.Chapters {
        if len(c.Pages) == 0 {
            continue
        }
        for _, p := range c.Pages {
            wg.Add(1)
            r := &dlPageReq{
                data: p,
                directory: c.Directory,
            }
            pg_queue <- r
        }
    }

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

