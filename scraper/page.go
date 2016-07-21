package scraper

import (
    "fmt"
)

type PageData struct {
    Url     string
    Image   *ImageData
}

type dlPageReq struct {
    data        *PageData
    directory   string
}

func NewPageData(url string) (*PageData) {
    return &PageData{Url: url}
}

func (p PageData) String() string {
    return fmt.Sprintf("<PageData Url:%q>", p.Url)
}

func (p *PageData) download() error {
    // download page html
    data, err := downloadThing(p.Url)
    if err != nil {
        return err
    }

    // get image url
    img_found := re_image.FindSubmatch(data)
    if len(img_found) < 1 {
        return fmt.Errorf("No page image found")
    }

    //fmt.Printf("Found image url: %s\n", img_found[1])
    p.Image = NewImageData(fmt.Sprintf("%s", img_found[1]))
    if err := p.Image.download(); err != nil {
        return err
    }
    return nil
}

