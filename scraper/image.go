package scraper

import (
    "fmt"
    "io/ioutil"
    "path"
    "strings"
)

type ImageData struct {
    Data    []byte
    Url     string
    Name    string
}

func (i ImageData) String() string {
    return fmt.Sprintf("<ImageData Name:%q Size:%d Url:%q>", i.Name, len(i.Data), i.Url)
}

func NewImageData(url string) (*ImageData) {
    _, filename := path.Split(url)
    if idx := strings.Index(filename, "?"); idx > -1 {
        filename = filename[:idx]
    }
    return &ImageData{Url: url, Name: filename}
}

func (i *ImageData) download() error {
    data, err := downloadThing(i.Url)
    if err != nil {
        return err
    }
    i.Data = data
    return nil
}

func (i *ImageData) writeToFile(directory string) error {
    if len(i.Data) == 0 {
        return fmt.Errorf("No data to write to file.")
    }

    fullpath := path.Join(directory, i.Name)
    return ioutil.WriteFile(fullpath, i.Data, 0655)
}

