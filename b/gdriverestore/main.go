package main

import (
  "os"
  "flag"
  "log"
  "path/filepath"
  "io"
  "encoding/json"
  "strings"
  "os/exec"

  "github.com/prasmussen/gdrive/gdrive"
  "github.com/boltdb/bolt"
)

type FileManifest struct {
  Path string
  GdriveId string
  Md5 string
}

func main() {
  verifyptr := flag.Bool("verify", false, "Use gdrivesync-state.boltdb to verify downloaded files")
  flag.Parse()
  verify := *verifyptr
  parentId := flag.Arg(0)
  if parentId == "" {
    log.Fatalln("usage: gdrivesync <parent_id>")
  }

  d, err := gdrive.New(os.Getenv("GDRIVE_CONFIG_DIR"), false, false)
  if err != nil {
    log.Fatalln("An error occurred creating Drive client: %v\n", err)
  }

  var db *bolt.DB
  if verify {
    db, err = bolt.Open("gdrivesync-state.boltdb", 0600, nil)
    if err != nil { log.Fatalln(err) }
  }


  var nextPageToken string

  for {
    caller := d.Children.List(parentId)
    if nextPageToken != "" {
      caller.PageToken(nextPageToken)
    }
    list, err := caller.Do()
    if err != nil { log.Fatalln(err) }

    for _, child := range list.Items {
      info, err := d.Files.Get(child.Id).Do()
      if err != nil { log.Fatalln(err) }
      if info.DownloadUrl == "" { continue }
      if info.Labels.Trashed { continue }

      path := info.Title

      var f *os.File

      if _, err = os.Stat(path); os.IsNotExist(err) {
        log.Println("downloading",path)

        // GET the download url
        res, err := d.Client().Get(info.DownloadUrl)

        err = os.MkdirAll(filepath.Dir(path), 0700)
        if err != nil { log.Fatalln("mkdir for",path,"returned",err) }

        f, err := os.Create(path+".gdriverestore-tmp")
        if err != nil { log.Fatalln(err) }

        _, err = io.Copy(f, res.Body)
        if err != nil { log.Fatalln(err) }
        err = res.Body.Close()
        if err != nil { log.Fatalln(err) }
        err = f.Close()
        if err != nil { log.Fatalln(err) }
      } else {
        log.Println("already found",path)
        f, err = os.Open(path)
        if err != nil { log.Fatalln(err) }
      }

      if verify {
        out2, err := exec.Command("md5sum", f.Name()).Output()
        if err != nil { log.Fatalln(err) }
        actual_md5 := strings.Fields(string(out2))[0]

        err = db.View(func(tx *bolt.Tx) error {
            v := tx.Bucket([]byte("GdriveManifestsByPath")).Get([]byte(path))
            var manifest FileManifest
            err := json.Unmarshal(v, &manifest)
            if err != nil { return err }

            if manifest.GdriveId != child.Id {
              log.Fatalln("ID mismatch for",path,"-- expected",manifest.GdriveId,"but got",child.Id)
            }

            if manifest.Md5 != actual_md5 {
              log.Fatalln("Md5 mismatch for",path,"-- expected",manifest.Md5,"but got",actual_md5)
            }

            return nil
        })
        if err != nil { log.Fatalln(err) }
      }

      // everything is well, move file into place
      // this will be a no-op if the file already existed
      err = os.Rename(f.Name(), path)
      if err != nil { log.Fatalln("mkdir for",path,"returned",err) }
    }

    nextPageToken = list.NextPageToken
    if nextPageToken == "" {
      break
    }
  }
}
