package scraper

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type Chapter struct {
    Number      float32
    Name        string
    Directory   string
    Url         string
    Pages       []*PageData
    downloaded  bool
}

var emptyChapter error = fmt.Errorf("Empty chapter")

func (c Chapter) String() string {
    return fmt.Sprintf("<Chapter Number:%0.1f Name:%q Directory:%q Pages:%d Url:%q>", c.Number, c.Name, c.Directory, len(c.Pages), c.Url)
}

func NewChapter(start_url, directory string) (*Chapter, error) {
    err := os.MkdirAll(directory, 0755)
    if err != nil {
        return nil, fmt.Errorf("Unable to Mkdir: %s", err)
    }

    return &Chapter{Url: start_url, Directory: directory}, nil
}

func (c *Chapter) addPage(url string) {
    for _, p := range c.Pages {
        if p.Url == url {
            return
        }
    }

    c.Pages = append(c.Pages, &PageData{Url: url})
}

func (c *Chapter) getPageUrls() error {
    raw_page, err := downloadThing(c.Url)
    if err != nil {
        return fmt.Errorf("Unable to download start page: %s", err)
    }

    urls := uniquify(getUrls(raw_page, c.Url))

    for _, u := range urls {
        if strings.Index(u, "html") > -1 {
            c.addPage(u)
        }
    }

    if len(c.Pages) == 0 {
        return emptyChapter
    }

    return nil
}

func (c *Chapter) rmdir() error {
    return os.Remove(c.Directory)
}

func (c *Chapter) rmr() error {
    return os.RemoveAll(c.Directory)
}

// Has anything been downloaded for the chapter?  Checks for the existence of
/// the chapter's zip file, and anything in its directory, in that order.
func (c *Chapter) emptyChapter() bool {
    if ex, _ := exists(c.Directory + ".zip"); ex {
        return false
    }

    items, _ := filepath.Glob(c.Directory + "/*")
    if len(items) == 0 {
        return true
    }
    return false
}

