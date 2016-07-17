package scraper

import (
    //"bytes"
    "fmt"
    "os"
    "strings"
)

type Chapter struct {
    Number      float32
    Name        string
    Directory   string
    Url         string
    Pages       []*PageData
}

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

func (c *Chapter) AddPage(url string) {
    for _, p := range c.Pages {
        if p.Url == url {
            return
        }
    }

    c.Pages = append(c.Pages, &PageData{Url: url})
}

func (c *Chapter) GetPageUrls() error {
    raw_page, err := downloadThing(c.Url)
    if err != nil {
        return fmt.Errorf("Unable to download start page: %s", err)
    }

    urls := uniquify(getUrls(raw_page, c.Url))

    for _, u := range urls {
        if strings.Index(u, "html") > -1 {
            c.AddPage(u)
        }
    }

    return nil
}

