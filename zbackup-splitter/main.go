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
  afterPartProgram := os.Args[3]

  i := 0

  eof := false
  for !eof {
    log.Println("commencing part ",i)
    partPath := backupPathPrefix+"_part"+strconv.Itoa(i)

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

    after := exec.Command(afterPartProgram, backupPathPrefix, strconv.Itoa(i))
    after.Stdout = os.Stdout
    after.Stderr = os.Stderr
    err = after.Run()
    if err != nil {
      log.Fatalln(err)
    }

    i++
  }
}
