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
  "time"

  _ "github.com/prasmussen/gdrive" // dummy so we can install the binary
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
  if f.IsDir() || strings.Contains(path, "AppleDouble") || path == "gdrivesync-state.boltdb" || path == "info" || strings.HasPrefix(path, "tmp/") {
    return nil
  }

  if *archiving && !strings.HasPrefix(path, "archived_bundles/") {
    return nil
  }
  fmt.Printf("Visited: %s\n", path)

  q := "'"+parent+"' in parents and title='"+path+"'"
  gdriveConfigDir := os.Getenv("GDRIVE_CONFIG_DIR")
  if gdriveConfigDir == "" { gdriveConfigDir = "~/.gdrive" }

  var manifest FileManifest

  err = db.View(func(tx *bolt.Tx) error {
    b := tx.Bucket([]byte("GdriveManifestsByPath"))
    if b == nil { return fmt.Errorf("didn't find bucket") }

    if v := b.Get([]byte(path)); v != nil {
      err := json.Unmarshal(v, &manifest)
      if err != nil { return err }
    }

    return nil
  })

  if err != nil {
    log.Println("uploading",path)
    cmd := exec.Command("haven-b-gdrive", "-c", gdriveConfigDir, "upload", "--parent", parent, "--file", path, "--title", path)
    err = cmd.Run()
    if err != nil { return err }
    out, err := exec.Command("haven-b-gdrive", "-c", gdriveConfigDir, "list", "-n", "-q", q).Output()
    if err != nil { return err }
    if string(out) == "" {
      log.Fatalln("uploaded file still doesn't exist")
    }

    id := strings.Fields(string(out))[0]
    log.Println("found",path,"at gdrive id",id)
    info, err := exec.Command("haven-b-gdrive", "-c", gdriveConfigDir, "info", "-i", id).Output()
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
  }

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

  if *archiving {
    // loop until there are no background changes
    archivingLoop()
  } else {
    // just do a single pass
    err = filepath.Walk(".", visit)
    if err != nil { log.Fatalln(err) }
  }
}


func archivingLoop() {
  exitfile := os.Getenv("EXIT_ON_ABSENT")

  for {
    err := filepath.Walk(".", visit)
    if err != nil { log.Fatalln(err) }

    _, err = os.Stat(exitfile)
    if err != nil {
      if !os.IsNotExist(err) {
        log.Println("could not stat exitfile",exitfile)
        log.Fatalln(err)
      }
      log.Println("exit file",exitfile,"is gone, doing one last sweep")
      err := filepath.Walk(".", visit)
      if err != nil { log.Fatalln(err) }
      log.Println("goodbye")
      return
    } else {
      log.Println("exit file",exitfile,"still present, sleeping for 5s before running again")
      time.Sleep(5*time.Second)
    }
  }
}
