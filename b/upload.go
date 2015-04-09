package main

import (
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "github.com/cenkalti/backoff"
  "github.com/pivotal-golang/bytefmt"

  "time"
  "encoding/json"
  "net/http"
  "io/ioutil"
  "bytes"
  "crypto/md5"
  "encoding/hex"
  "math"
  "log"
  "fmt"
  "os/user"
  "io"
  "os"
  "strconv"
)

// use 16M chunk size .. ?
const PreferredChunkSize = 1024 * 1024 * 1

func main() {
  zero, _ := os.Open("/dev/urandom")
  id := upload(&io.LimitedReader{N: 1024*1024*2, R: zero}, "zerotest")
  // output just the ID
  fmt.Printf("%s\n", id)
}

/// Returns the Google Drive ID of the created file on success, panics on error.
func upload(in_raw io.Reader, filename string) string {
  counter := io.LimitedReader{N: math.MaxInt64, R: in_raw}
  hash := md5.New()
  in := io.TeeReader(&counter, hash)
  client, err := getAuthenticatedClient()
  if err != nil { log.Fatalln(err)}

  var metadataJson = []byte(`{"title":"Buy cheese and bread for breakfast."}`)

  // Don't bother with retrying this first request; if the network is down, we can safely fail
  // and we probably have failed at the OAuth stage already.
  r, err := client.Post("https://www.googleapis.com/upload/drive/v2/files?uploadType=resumable",
    "application/json; charset=UTF-8",
    bytes.NewBuffer(metadataJson))
  if err != nil { log.Fatalln(err)}
  if r.StatusCode != http.StatusOK { log.Fatalln("failed to create upload") }
  uploadUrl := r.Header.Get("Location")
  log.Println("Metadata sent, starting upload...")

  var chunk bytes.Buffer
  var slop bytes.Buffer
  chunk.Grow(PreferredChunkSize)

  // A word on sizes: the total sizes are int64 for .. you know, scalability.
  // However, buffer and chunk sizes are ints, as that's what golang uses for
  // memory management. Think twice if you feel like you have to cast
  // int64->int somewhere.
  // TODO: size_t is likely unsigned, should we settle for uint64 as well?

  // The current estimate for the total size of the input.
  // After the following loop terminates, it will be accurate.
  var total_size int64

  // The byte offset of byte following the one that was last sent to the server.
  var chunkoffs int64

  for {
    chunk.Reset()
    chunksize, err := io.Copy(&chunk, &io.LimitedReader{N: PreferredChunkSize, R: in})
    if err != nil { log.Fatalln(err) }
    total_size += chunksize
    if chunksize != PreferredChunkSize {
      // Short read; that means we've reached EOF (since err is nil).
      // The routine after this loop will take care of sending this short-sized chunk.
      slop = chunk
      break
    }

    done_in_chunk := 0
    nextoffs := chunkoffs + int64(done_in_chunk)
    req, err := http.NewRequest("PUT", uploadUrl, bytes.NewReader(chunk.Bytes()[done_in_chunk:]))
    req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", nextoffs, chunkoffs+int64(len(chunk.Bytes())-1)))
    resp, err := doWithRetry(client,req,308)
    if err != nil { log.Fatalln(err)}
    if resp.StatusCode != 308 { log.Fatalln("failed to upload chunk") }
    var reported_start int64
    var reported_end int64
    n, err := fmt.Sscanf(resp.Header.Get("Range"), "bytes=%d-%d", &reported_start, &reported_end)
    if err != nil || n != 2 {
      log.Fatalln("failed to parse Range header",resp.Header.Get("Range"))
    }
    if reported_start != 0 {
      log.Fatalln("insane reported_start, expected",0,"got",reported_start)
    }
    if reported_end != chunkoffs+int64(chunksize-1) {
      // let's not deal with partially uploaded chunks for now
      log.Fatalln("insane reported_end, expected",chunkoffs+chunksize-1,"got",reported_end)
    }

    s := fmt.Sprintf("(%s in)", bytefmt.ByteSize(uint64(chunkoffs+chunksize)))
    log.Println("Uploaded chunk",chunkoffs,"--",chunkoffs+chunksize,s)
    chunkoffs += chunksize
  }

  log.Println("Reached end of input, finalizing..")

  // There might be some leftovers from the next-to-last chunk that we have to
  // flush now, they are stored in `slop`.
  // Now send the final chunk including the total size. Google will complain if we
  // screwed up the ranges somewhere in between.
  req, err := http.NewRequest("PUT", uploadUrl, &slop)
  var bytes_here string
  if slop.Len() == 0 {
    bytes_here = "*"
  } else {
    log.Println("Uploading",slop.Len(),"remaining bytes..")
    bytes_here = fmt.Sprintf("%d-%d",total_size-int64(slop.Len()),total_size-1)
  }
  req.Header.Set("Content-Range", fmt.Sprintf("bytes %s/%d", bytes_here, total_size))
  resp, err := doWithRetry(client,req,200)
  if err != nil { log.Fatalln(err)}
  if resp.StatusCode != 200 {
    x, err := ioutil.ReadAll(resp.Body)
    if err != nil { log.Fatalln(err)}
    log.Fatalln("failed to finalize upload", string(x))
  }
  outputMetadataJson, err := ioutil.ReadAll(resp.Body)
  //log.Println(string(outputMetadataJson))
  if err != nil { log.Fatalln(err)}
  var meta GdriveFileMeta
  err = json.Unmarshal(outputMetadataJson, &meta)
  if err != nil { log.Fatalln(err)}
  log.Println("OK, Google has all",total_size,"bytes")


  // Sanity check 1:
  // Make sure we have actually drained the input.
  read := math.MaxInt64 - counter.N
  if read != total_size {
    log.Fatalln("whoops -- uploaded",total_size,"bytes, but read",read)
  }
  var tmpbuf bytes.Buffer
  n, err := counter.Read(tmpbuf.Bytes())
  if n != 0 {
    log.Fatalln("didn't drain input buffer -- read",n,"instead of 0")
  }
  if err != io.EOF {
    log.Fatalln("didn't drain input buffer --",err,"is not EOF")
  }


  // Sanity check 2:
  // Check the md5sum of the gdrive file against what Google tells us
  expected_md5 := hex.EncodeToString(hash.Sum(nil))
  actual_md5 := meta.Md5

  if expected_md5 != actual_md5 {
    log.Fatalln("final md5 check failed: expected",expected_md5,"but Google has",actual_md5)
  } else {
    log.Println("OK, MD5 check passed --",expected_md5)
  }

  if meta.Id == "" {
    log.Fatalln("file doesn't have an ID!")
  }

  if meta.Size != strconv.FormatInt(total_size, 10) {
    log.Fatalln("reported file size is",meta.Size," -- but expected to have",total_size)
  }

  return meta.Id
}

type GdriveFileMeta struct {
  Md5 string `json:"md5Checksum"`
  Size string `json:"fileSize"`
  Id string `json:"id"`
}

// Our backoff parameters, chosen to allow grotesquely long intervals.
// It's not like anyone else cares (or should care!) -- if you pipe `zfs send` or `tar`
// into this, they can wait as long as needed.
var timer = &backoff.ExponentialBackOff {
  InitialInterval:     500 * time.Millisecond,
  RandomizationFactor: 0.5,
  Multiplier:          1.5,
  MaxInterval:         1 * time.Minute,
  MaxElapsedTime:      24 * time.Hour, // Google deletes the temp state after 1d, so it doesn't make much sense to wait longer
  Clock:               backoff.SystemClock,
}

func doWithRetry(client *http.Client, req *http.Request, expectedStatus int) (*http.Response, error) {
  var resp *http.Response
  i := 0
  err := backoff.Retry(func() error {
    i++
    var int_err error
    resp, int_err = client.Do(req)
    if int_err == nil && resp.StatusCode != expectedStatus {
      int_err = fmt.Errorf("expected status %d, but got %d", expectedStatus, resp.StatusCode)
    }
    if int_err != nil {
      log.Println("Attempt",i,"failed:",int_err)
    }
    return int_err
  }, timer)

  if err != nil {
    return nil, err
  } else {
    return resp, nil
  }
}


func getAuthenticatedClient() (*http.Client, error) {
  config := &oauth2.Config{
    ClientID:     "876823704581-jiqb6gvjt77ea7cevsqsf6jr4on0b5f5.apps.googleusercontent.com",
    ClientSecret: "lERovWunL6URp0GlG5bgL9cS",
    Scopes:       []string{"https://www.googleapis.com/auth/drive"},
    RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
    Endpoint: google.Endpoint,
  }

  tok := new(oauth2.Token)

  usr, err := user.Current()
  if err != nil { return nil, err }
  var cachePath = usr.HomeDir + "/.haven-b-gdrivecreds"

  // try reading from the cache
  if cacheData, err := ioutil.ReadFile(cachePath); err == nil {
    err := json.Unmarshal(cacheData, &tok)
    if err != nil {
      log.Println("Found malformed data in cache file",cachePath)
    }
  }

  // Interactively refresh the token if not present
  // (don't use tok.Valid() since it is also false if we just have to refresh the token)
  if tok.RefreshToken == "" {
    // Redirect user to consent page to ask for permission
    // for the scopes specified above.
    url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
    fmt.Printf("Visit the URL for the auth dialog: %v", url)

    // Use the authorization code that is pushed to the redirect URL.
    // NewTransportWithCode will do the handshake to retrieve
    // an access token and initiate a Transport that is
    // authorized and authenticated by the retrieved token.
    var code string
    if _, err := fmt.Scan(&code); err != nil {
        log.Fatal(err)
    }
    tok, err = config.Exchange(oauth2.NoContext, code)
    if err != nil {
        log.Fatal(err)
    }

    marshaled, err := json.Marshal(tok)
    if err != nil { return nil, err }
    err = ioutil.WriteFile(cachePath, marshaled, 0400)
    if err != nil { return nil, err }
  }

  if tok.RefreshToken == "" {
    return nil, fmt.Errorf("failed to obtain a valid token")
  }

  client := config.Client(oauth2.NoContext, tok)

  return client, nil
}
