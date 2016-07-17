package scraper

import (
    "bytes"
    //"fmt"
    "io/ioutil"
    "net/http"
    "regexp"
    "strings"
    "time"
)

const mh_string string = "www.mangahere.co/manga/"

var client = &http.Client{Timeout: time.Second * 10}

var re_image = regexp.MustCompile(`src="(http://a.mhcdn.net/store/manga[^"]+)"`)   // "
var re_url = regexp.MustCompile(`"(http://[^"]+)"`) //" // Because vim on windows is dumb, apparently
var re_urlch = regexp.MustCompile(`/c(\d+\.?\d*)/`) //

func downloadThing(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:36.0) Gecko/20100101 Firefox/36.0")

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    return ioutil.ReadAll(resp.Body) 
}

func getUrls(html []byte, baseurl string) []string {
    var s []string
    base := []byte(baseurl)
    matches := re_url.FindAllSubmatch(html, -1)

    if len(matches) < 2 {
        return nil
    }

    for _, m := range matches {
        if len(m) >= 2 {
            if bytes.Index(m[1], base) > -1 {
                s = append(s, string(m[1]))
            }
        }
    }

    return s
}

func dump(filename string, data []byte) error {
    return ioutil.WriteFile(filename, data, 0655)
}

func uniquify(things []string) []string {
    if len(things) == 0 {
        return []string{}
    }

    m := make(map[string]bool)
    for _, thing := range things {
        m[thing] = true
    }

    var u []string
    for key, _ := range m {
        u = append(u, key)
    }
    return u
}

func prettifyName(input string) string {
    return strings.ToTitle(strings.Replace(input, "_", " ", -1))
}

