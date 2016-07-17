package scraper

import (
    "archive/zip"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
)

type zipFile struct {
    path    string      // eg "manga/<manga_name>/<Manga> <chapter>.zip"
    files   []string    // full path of images for the chapter
}

func (z zipFile) String() string {
    fs := []string{}
    for _, f := range z.files {
        fs = append(fs, fmt.Sprintf("%q", f))
    }
    return fmt.Sprintf("<zipFile path:%q files:['%s']>", z.path, strings.Join(fs, "', '"))
}

func (s *Series) ZipChapters() error {
    var zips []zipFile
    root := s.getDir()

    for _, c := range s.Chapters {
        z := zipFile{
            path: root + "/" + c.Name + ".zip",
            files: []string{},
        }

        files, err := filepath.Glob(c.Directory + "/*.*")
        if err != nil {
            return fmt.Errorf("Error globbing: %s", err)
        }

        for _, f := range files {
            if f == "." || f == ".." {
                continue
            }
            f = strings.Replace(f, "\\\\", "/", -1)
            z.files = append(z.files, f)
        }
        if len(z.files) > 0 {
            zips = append(zips, z)
        }
    }

    nd := true
    pr_count = 0
    pr_wg.Add(1)
    go printProgress(&nd, len(zips), "Zip")

    for _, z := range zips {
        printProgressAdd(1)
        if err := z.Create(); err != nil {
            return fmt.Errorf("Error creating zipFile: %s", err)
        }
    }

    nd = false
    pr_wg.Wait()
    fmt.Println("Finished zipping up chapters")

    return nil
}

func (z *zipFile) Create() error {
    buf, err := os.Create(z.path)
    if err != nil {
        return err
    }
    defer buf.Close()

    w := zip.NewWriter(buf)
    defer w.Close()

    for _, file := range z.files {
        // open image file
        imgFile, err := os.Open(file)
        if err != nil {
            return err
        }
        defer imgFile.Close()

        // read the entire file
        data, err := ioutil.ReadAll(imgFile)
        if err != nil {
            return err
        }

        // stat the file to create the file's zip header
        stat, err := imgFile.Stat()
        if err != nil {
            return err
        }
        imgFile.Close()

        // If we just use zip.Create() the header isn't created.
        header, err := zip.FileInfoHeader(stat)
        if err != nil {
            return err
        }

        // Overwrite whatever's there with what we want.
        header.Name = filepath.Base(file)

        f, err := w.CreateHeader(header)
        if err != nil {
            return err
        }

        // Write the data to disk
        if _, err = f.Write(data); err != nil {
            return err
        }
        w.Flush()
    }

    return nil
}

