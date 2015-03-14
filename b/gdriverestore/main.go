package main

import (
  "os"
  "flag"
  "log"
  "path/filepath"
  "io"

  "github.com/prasmussen/gdrive/gdrive"
)

func main() {
  verifyptr := flag.Bool("verify", false, "Use gdrivesync-state.boltdb to verify downloaded files")
  flag.Parse()
  verify := *verifyptr
  parentId := flag.Arg(0)
  if parentId == "" {
    log.Fatalln("usage: gdrivesync <parent_id>")
  }

  if verify {
    err := runFromManifest()
    if err != nil {
      log.Fatalln(err)
    } else {
      return
    }
  }

  var nextPageToken string

  for {
    d, err := gdrive.New(os.Getenv("GDRIVE_CONFIG_DIR"), false, false)
    if err != nil {
      log.Fatalln("An error occurred creating Drive client: %v\n", err)
    }
    caller := d.Children.List(parentId)
    if nextPageToken != "" {
      caller.PageToken(nextPageToken)
    }
    list, err := caller.Do()
    if err != nil { log.Fatalln(err) }

    for _, child := range list.Items {
      d, err := gdrive.New(os.Getenv("GDRIVE_CONFIG_DIR"), false, false)
      if err != nil {
        log.Fatalln("An error occurred creating Drive client: %v\n", err)
      }
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
        if err != nil { log.Fatalln(err) }
      }

      // everything is well, move file into place
      // this will be a no-op if the file already existed
      err = os.Rename(f.Name(), path)
      if err != nil { log.Fatalln("mkdir for",path,"returned",err) }

      if os.Getenv("HAVEN_B_CRASHAT") == "onerestore" { crash() }
    }

    nextPageToken = list.NextPageToken
    if nextPageToken == "" {
      break
    }
  }
}

func crash() {
  self, err := os.FindProcess(os.Getpid())
  if err != nil { log.Fatalln(err) }
  self.Signal(os.Kill)
}
