package main

import (
  "os"
  "os/exec"
  "io"
  "log"
  "strconv"
)

func main() {
  partSize, err := strconv.ParseInt(os.Args[1], 10, 64)
  if err != nil {
    log.Fatalln(err)
  }
  backupPathPrefix := os.Args[2]

  i := 0

  os.Exit(0)

  eof := false
  for !eof {
    partPath := backupPathPrefix+"_part"+strconv.Itoa(i)
    log.Println(partPath)

    rd, wr := io.Pipe()
    cmd := exec.Command("zbackup", "backup", partPath)
    cmd.Stdin = rd
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Start()
    if err != nil {
      log.Fatalln(err)
    }

    _, err = io.CopyN(wr, os.Stdin, partSize)
    if err != nil {
      if err == io.EOF {
        eof = true
      } else {
        log.Fatalln(err)
      }
    }

    wr.Close()

    err = cmd.Wait()

    if err != nil {
      log.Fatalln(err)
    }

    i++
  }
}
