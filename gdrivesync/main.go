package main

import (
  "path/filepath"
  "os"
  "flag"
  "fmt"
  "log"
  "os/exec"
  "strings"
  "regexp"
  "encoding/json"

  "github.com/boltdb/bolt"
)

var parent string

var infoMd5Exp = regexp.MustCompile("Md5sum: ([0-9a-f]+)")

type FileManifest struct {
  Path string
  GdriveId string
  Md5 string
}

func visit(path string, f os.FileInfo, err error) error {
  if f.IsDir() || strings.Contains(path, "AppleDouble") || path == "gdrivesync-state.boltdb" || path == "info" {
    return nil
  }

  if *archiving && !strings.HasPrefix(path, "archived_bundles/") {
    return nil
  }
  fmt.Printf("Visited: %s\n", path)

  q := "'"+parent+"' in parents and title='"+path+"'"
  out, err := exec.Command("gdrive", "list", "-n", "-q", q).Output()
  if err != nil { return err }
  if string(out) == "" {
    log.Println("uploading",path)
    cmd := exec.Command("gdrive", "upload", "--parent", parent, "--file", path, "--title", path)
    err = cmd.Run()
    if err != nil { return err }
    out, err = exec.Command("gdrive", "list", "-n", "-q", q).Output()
    if err != nil { return err }
    if string(out) == "" {
      log.Fatalln("uploaded file still doesn't exist")
    }
  }

  id := strings.Fields(string(out))[0]
  log.Println("found",path,"at gdrive id",id)
  info, err := exec.Command("gdrive", "info", "-i", id).Output()
  if err != nil { return err }
  online_md5 := infoMd5Exp.FindStringSubmatch(string(info))[1]

  out2, err := exec.Command("md5sum", path).Output()
  if err != nil { return err }
  actual_md5 := strings.Fields(string(out2))[0]

  if online_md5 != actual_md5 {
    log.Println("found md5",online_md5,"while actual_md5 is",actual_md5)
    log.Fatalln("md5 mismatch")
  }

  // add it to our manifest
  err = db.Update(func(tx *bolt.Tx) error {
    m := FileManifest{Path: path, GdriveId: id, Md5: actual_md5}
    b := tx.Bucket([]byte("GdriveManifestsByPath"))

    x, err := json.Marshal(m)
    if err != nil { return err }
    err = b.Put([]byte(path), x)
    return err
  })
  if err != nil { log.Fatalln(err) }

  if *archiving {
    log.Println("removing local copy of",path,"after successful upload")
    // the file is uploaded, we can remove it locally
    err = os.Remove(path)
    if err != nil { log.Fatalln(err) }
  }

  return nil
}

var db *bolt.DB
var archiving *bool

func main() {
  archiving = flag.Bool("archive-and-delete", false, "Only visit archived_bundles/, but delete files after archiving")
  flag.Parse()
  parent = flag.Arg(0)
  if parent == "" {
    log.Fatalln("usage: gdrivesync <parent>")
  }

  var err error
  db, err = bolt.Open("gdrivesync-state.boltdb", 0600, nil)
  if err != nil { log.Fatalln(err) }

  err = db.Update(func(tx *bolt.Tx) error {
    _, err := tx.CreateBucketIfNotExists([]byte("GdriveManifestsByPath"))
    return err
  })
  if err != nil { log.Fatalln(err) }

  err = filepath.Walk(".", visit)
  if err != nil { log.Fatalln(err) }
}
