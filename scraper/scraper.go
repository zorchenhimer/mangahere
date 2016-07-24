package scraper

import (
    "fmt"
    "os"
    "path/filepath"
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
    start_idx   int     // chapter index to start on
    Force       bool    // Force downloading of chapters
}

var num_dl_threads int = 4
const retry_count int = 4

func NewSeries(url string) (*Series, error) {
    // validate url
    u := strings.Split(strings.Trim(url, "/"), "/")
    if len(u) < 5 || u[2] != "www.mangahere.co" {
        return nil, fmt.Errorf("Invalid mangahere url: %q", strings.Join(u, "/"))
    }

    if u[3] != "manga" {
        return nil, fmt.Errorf("Not a manga url: %q", u[3])
    }

    // This is a chapter or page url, not a detail url.  Save the chapter and
    // we'll handle this in a little bit.
    start_chapter := ""
    if len(u) > 5 {
        start_chapter = u[5]
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
        start_idx:  0,
    }

    // Get the chapters
    det_urls := getUrls(manga_det_html, s.Url)
    sort.Strings(det_urls)
    for idx, u := range det_urls {
        if re_urlch.MatchString(u) {
            s.addChapter(u)
        }

        // If we were passed a chapter url earlier, and it matches the current
        // one, ask if we should start at the given chapter.
        if idx > 0 {
            if u == s.Url + start_chapter + "/" {
                ans := yesNoPrompt(fmt.Sprintf("Start from chapter %q? ", start_chapter), Yes)
                if ans == Yes {
                    s.start_idx = idx - 1
                }
            }
        }
    }

    return s, nil
}

// SetBaseDir() will set the base directory that the manga will be downloaded
// to.  This defaults to "manga".
// FIXME: Make baseDir absolute?
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

// Cleanup() will remove any chapter folders that have already been zipped up.
// It will also remove any empty chapter folders.
func (s *Series) Cleanup() error {
    for _, c := range s.Chapters {
        if itm, err := filepath.Glob(c.Directory + "/*"); err == nil && len(itm) == 0 {
            err = c.rmdir()
            if err != nil {
                // Only display an error if it isn't a "this directory doesn't exist" type error
                if !os.IsNotExist(err) {
                    fmt.Printf("Error removing 'empty' chapter directory: %s", err)
                }
            }
            continue
        }
        if ex, _ := exists(c.Directory + ".zip"); ex == false {
            continue
        }
        if err := c.rmr(); err != nil {
            return err
        }
    }

    return nil
}

// Add a chapter to a series from its URL. Note that this does not download the
// chapter, it only creates the object.
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

// chapter download worker goroutine
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
        c.downloaded = true
        printProgressAdd(1)
        wg.Done()
    }
}

// page download worker goroutine
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

        // Retrying here would probably have no effect whatsoever on the
        // outcome (insufficient privileges, etc), so don't bother.
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
    for _, c := range s.Chapters[s.start_idx:] {
        // Download chapter if the directory is empty, or if the force flag is
        // given at the command line.
        if c.emptyChapter() || s.Force {
            wg.Add(1)
            ch_queue <- c
        }
        fmt.Println("")
    }
    nd := true
    pr_wg.Add(1)
    go printProgress(&nd, len(s.Chapters) - s.start_idx, "Chapter")

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
    length := 0

    for _, c := range s.Chapters[s.start_idx:] {
        if !c.downloaded || len(c.Pages) == 0 {
            continue
        }
        for _, p := range c.Pages {
            wg.Add(1)
            r := &dlPageReq{
                data: p,
                directory: c.Directory,
            }
            pg_queue <- r
            length++
        }
    }

    nd = true
    pr_wg.Add(1)
    go printProgress(&nd, length, "Page")

    wg.Done()
    wg.Wait()
    close(pg_queue)
    nd = false
    pr_wg.Wait()

    fmt.Println("Finished DLing pages")
}

