package main

import (
  "github.com/boltdb/bolt"
  "fmt"
  "os"
  "os/exec"
  "encoding/json"
  "strings"
  "path/filepath"
  "github.com/prasmussen/gdrive/gdrive"
  "log"
  "io"
)

type FileManifest struct {
  Path string
  GdriveId string
  Md5 string
}

func ensureFile(manifest FileManifest) error {
  path := manifest.Path

  var f *os.File

  if _, err := os.Stat(path); os.IsNotExist(err) {
    log.Println("downloading",path)
    d, err := gdrive.New(os.Getenv("GDRIVE_CONFIG_DIR"), false, false)
    if err != nil {
      log.Fatalln("An error occurred creating Drive client: %v\n", err)
    }

    info, err := d.Files.Get(manifest.GdriveId).Do()
    if err != nil { log.Fatalln(err) }

    // GET the download url
    res, err := d.Client().Get(info.DownloadUrl)

    err = os.MkdirAll(filepath.Dir(path), 0700)
    if err != nil { log.Fatalln("mkdir for",path,"returned",err) }

    f, err = os.Create(path+".gdriverestore-tmp")
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
    if err != nil { return err }
  }

  md5_out, err := exec.Command("md5sum", f.Name()).Output()
  if err != nil { log.Fatalln(err) }
  actual_md5 := strings.Fields(string(md5_out))[0]

  if manifest.Md5 != actual_md5 {
    return fmt.Errorf("Md5 mismatch for %s -- expected %s but got %s", path, manifest.Md5,actual_md5)
  }

  // everything is well, move file into place
  // this will be a no-op if the file already existed
  err = os.Rename(f.Name(), path)
  if err != nil { log.Fatalln("mkdir for",path,"returned",err) }

  if os.Getenv("HAVEN_B_CRASHAT") == "onerestore" { crash() }

  return nil
}

func runFromManifest() error {
  db, err := bolt.Open("gdrivesync-state.boltdb", 0600, nil)
  if err != nil { return err }

  return db.View(func(tx *bolt.Tx) error {
    b := tx.Bucket([]byte("GdriveManifestsByPath"))
    if b == nil { return fmt.Errorf("didn't find bucket") }
    c := b.Cursor()

    for k, v := c.First(); k != nil; k, v = c.Next() {
        var manifest FileManifest
        err := json.Unmarshal(v, &manifest)
        if err != nil { return err }
        err = ensureFile(manifest)
        if err != nil { return err }
    }

    return nil
  })
}
