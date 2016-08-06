package scraper

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "regexp"
    "strings"
    "sync"
    "time"
)

type YesNo bool

const Yes YesNo = true
const No  YesNo = false

const mh_string string = "www.mangahere.co/manga/"

// The client used for downloading things.
var client = &http.Client{Timeout: time.Second * 10}

var re_image = regexp.MustCompile(`src="(http://[^"]+\.mhcdn\.net/store/manga[^"]+)"`)   // "
var re_url = regexp.MustCompile(`"(http://[^"]+)"`) //" // Because vim on windows is dumb, apparently
var re_urlch = regexp.MustCompile(`/c(\d+\.?\d*)/`) //

// Download a thing. All download requests go through here.
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

// Get all the urls on the page that contain baseurl.
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

// This doesn't have much use yet.  This version of the name isn't used anywhere.
func prettifyName(input string) string {
    return strings.ToTitle(strings.Replace(input, "_", " ", -1))
}

var pr_count int
var pr_lock sync.Mutex
var pr_wg sync.WaitGroup

// This should only be called as a goroutine.
func printProgress(notdone *bool, total int, prefix string) {
    lastLen := 0
    wenotdone := true

    for *notdone || wenotdone {
        
        // The lock here probably isn't needed, but it can't really hurt too
        // much.
        pr_lock.Lock()
        text := fmt.Sprintf("%s % 4d/%d", prefix, pr_count, total)
        pr_lock.Unlock()

        lastLen = len(text)
        fmt.Printf(text)
        
        time.Sleep(time.Second / 8)

        // Having this here instead of at the start of the loop means we can
        // cleanup after ourselves.
        for i := 0; i < lastLen; i++ {
            fmt.Printf("\b")
        }

        // Block until we're all done.
        if *notdone == false {
            wenotdone = false
        }
    }
    pr_wg.Done()
}

// Update the progress counter
func printProgressAdd(delta int) {
    pr_lock.Lock()
    pr_count += delta
    pr_lock.Unlock()
}

// exists returns whether the given file or directory exists or not.
// Taken from https://stackoverflow.com/a/10510783
func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return true, err
}

func yesNoPrompt(prompt string, default_option YesNo) YesNo {
    if default_option == Yes {
        prompt += " [Y/n] "
    } else {
        prompt += " [y/N] "
    }

    for {
        input := ""
        fmt.Print(prompt)
        fmt.Scanln(&input)

        if len(input) == 0 {
            return default_option
        }

        switch strings.ToLower(input) {
            case "y", "yes":
                return Yes
            case "n", "no":
                return No
        }
    }
}

