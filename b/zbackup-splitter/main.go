package main

import (
  "os"
  "os/exec"
  "io"
  "log"
  "strconv"
  "path/filepath"
  "strings"
)

func main() {
  targetedPartSize, err := strconv.ParseInt(os.Args[1], 10, 64)
  if err != nil {
    log.Fatalln(err)
  }
  backupPathPrefix := os.Args[2]
  afterPartProgram := os.Args[3]

  i := 0

  eof := false
  for !eof {
    next_n := targetedPartSize

    log.Println("commencing part ",i)
    partPath := backupPathPrefix+"_part"+strconv.Itoa(i)

    rd, wr := io.Pipe()
    cmd := exec.Command("haven-b-zbackup", "--password-file", os.Getenv("PWFILE"), "backup", partPath)
    cmd.Stdin = rd
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Start()
    if err != nil {
      log.Fatalln(err)
    }

    zbackup := make(chan error)
    go func() {
      zbackup <- cmd.Wait()
    }()
    ioerr := make(chan error)

    part_loop: for {
      log.Println("feeding",next_n)

      go func() {
        _, err := io.CopyN(wr, os.Stdin, next_n)
        ioerr <- err
      }()

      select {
      case err := <-zbackup:
        log.Fatalln("zbackup exited unexpectedly:",err)
      case err := <-ioerr:
        if err != nil {
          if err == io.EOF {
            log.Println("input EOF")
            eof = true
            break part_loop
          } else {
            log.Fatalln(err)
          }
        }
      }

      out, err := exec.Command("du", "-s", "-k", filepath.Dir(backupPathPrefix)+"/../tmp").Output()
      if err != nil {
        log.Fatalln(err)
      }
      tmpSize, err := strconv.Atoi(strings.Fields(string(out))[0])
      if err != nil {
        log.Fatalln(err)
      }
      tmpSize = tmpSize * 1000
      log.Println("tmp/ now at: ",tmpSize,"target", targetedPartSize)

      next_n = targetedPartSize - int64(tmpSize)
      if next_n <= 0 {
        break
      }
      next_n *= 2
      if next_n < 256*1024 {
        next_n = 1024*1024
      }
    }

    log.Println("part",i,"is complete")

    // commit this part to zbackup and wait for it to exit
    wr.Close()
    err = <-zbackup

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
