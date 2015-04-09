package main

import (
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "encoding/json"
  "net/http"
  "io/ioutil"
  "bytes"

  "log"
  "fmt"
  "os/user"
  "io"
  "os"
)

func main() {
  zero, _ := os.Open("/dev/zero")
  upload(&io.LimitedReader{N: 1024*1024*3, R: zero}, "zerotest")
}

func upload(in io.Reader, filename string) {
  client, err := getAuthenticatedClient()
  if err != nil { log.Fatalln(err)}

  var metadataJson = []byte(`{"title":"Buy cheese and bread for breakfast."}`)

  r, err := client.Post("https://www.googleapis.com/upload/drive/v2/files?uploadType=resumable",
    "application/json; charset=UTF-8",
    bytes.NewBuffer(metadataJson))
  if err != nil { log.Fatalln(err)}
  if r.StatusCode != http.StatusOK { log.Fatalln("failed to create upload") }
  uploadUrl := r.Header.Get("Location")
  log.Println("upload is", uploadUrl)

  // use 16M chunk size .. ?
  const ChunkSize = 1024 * 1024 // * 16

  var chunk bytes.Buffer
  chunk.Grow(ChunkSize)
  total_size := 0

  chunkoffs := 0
  for {
    chunk.Reset()
    chunksize_64, err := io.Copy(&chunk, &io.LimitedReader{N: ChunkSize, R: in})
    if err != nil { log.Fatalln(err) }
    chunksize := int(chunksize_64) // sigh..
    total_size += chunksize
    if chunksize == 0 {
      log.Println("EOF!")
      break
    }

    done_in_chunk := 0
    nextoffs := chunkoffs + done_in_chunk
    req, err := http.NewRequest("PUT", uploadUrl, bytes.NewReader(chunk.Bytes()[done_in_chunk:]))
    req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", nextoffs, chunkoffs+len(chunk.Bytes())-1))
    resp, err := client.Do(req)
    log.Println(resp, err)
    if err != nil { log.Fatalln(err)}
    if resp.StatusCode != 308 { log.Fatalln("failed to upload chunk") }
    reported_start := 0
    reported_end := 0
    n, err := fmt.Sscanf(resp.Header.Get("Range"), "bytes=%d-%d", &reported_start, &reported_end)
    if err != nil || n != 2 {
      log.Fatalln("failed to parse Range header",resp.Header.Get("Range"))
    }
    if reported_start != 0 {
      log.Fatalln("insane reported_start, expected",0,"got",reported_start)
    }
    if reported_end != chunkoffs+chunksize-1 {
      // let's not deal with partially uploaded chunks for now
      log.Fatalln("insane reported_end, expected",chunkoffs+chunksize-1,"got",reported_end)
    }

    log.Println("uploaded da chunk")
    chunkoffs += chunksize
  }

  log.Println("reached EOF, finishing..")

  // sanity check: validate that we have everything by querying the current range one last time
  req, err := http.NewRequest("PUT", uploadUrl, nil)
  req.Header.Set("Content-Range", "bytes */*")
  resp, err := client.Do(req)
  if resp.StatusCode != 308 { log.Fatalln("failed to validate final upload") }
  reported_start := 0
  reported_end := 0
  n, err := fmt.Sscanf(resp.Header.Get("Range"), "bytes=%d-%d", &reported_start, &reported_end)
  if err != nil || n != 2 {
    log.Fatalln("failed to parse Range header",resp.Header.Get("Range"))
  }
  if reported_start != 0 {
    log.Fatalln("insane reported_start, expected",0,"got",reported_start)
  }
  // check that the entire file was uploaded
  // TODO: what about empty files?
  if reported_end != total_size-1 {
    log.Fatalln("insane reported_end, expected",total_size-1,"got",reported_end)
  }

  // now actually finish the upload
  req, err = http.NewRequest("PUT", uploadUrl, nil)
  req.Header.Set("Content-Length", "0")
  req.Header.Set("Content-Range", fmt.Sprintf("bytes */%d", total_size))
  resp, err = client.Do(req)
  if err != nil { log.Fatalln(err)}
  if resp.StatusCode != 200 { log.Fatalln("failed to finalize upload") }
  outputMetadataJson, err := ioutil.ReadAll(resp.Body)
  log.Println(string(outputMetadataJson))
  log.Println("yep, google has all",total_size,"bytes")
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
