package main

import (
  //"github.com/cenkalti/backoff"
  "github.com/docopt/docopt-go"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"

  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "os/user"
)

// use 16M chunk size .. ?
const PreferredChunkSize = 1024 * 1024 * 16

var client *http.Client

func main() {
  usage := `Download a file from Google Drive to stdout.

Usage:
  haven-b-download <gdrive_id>

Options:
  --name        Use gdrive_id as a file name, not an ID.
  -h --help     Show this screen.
  --version     Show version.`

  arguments, err := docopt.Parse(usage, nil, true, "0.1", false)
  id := arguments["<gdrive_id>"].(string)
  if id == "" {
    log.Fatalln("invalid arguments")
  }
  if err != nil {
    log.Fatalln(err)
  }

  // Try to get a client here. This will start the OAuth2 flow.
  // Do this before doing the terminal check so that we *can* use a terminal for this.
  client, err = getAuthenticatedClient()
  if err != nil {
    log.Fatalln(err)
  }


//   q := fmt.Sprintf("title='%s'", id)
//   https://www.googleapis.com/drive/v2/files?query=
//   req, err := http.NewRequest("GET", "http://example.com", nil)
// // ...
// req.Header.Add("If-None-Match", `W/"wyzzy"`)
// resp, err := client.Do(req)

  resp, err := client.Get(fmt.Sprintf("https://www.googleapis.com/drive/v2/files/%s?alt=media", id))
  if err != nil {
    log.Fatalln(err)
  }
  if resp.StatusCode != 200 {
    log.Fatalln("got status",resp.StatusCode)
  }

  _, err = io.Copy(os.Stdout, resp.Body)
  if err != nil {
    log.Fatalln(err)
  }
}

func getAuthenticatedClient() (*http.Client, error) {
  config := &oauth2.Config{
    ClientID:     "876823704581-jiqb6gvjt77ea7cevsqsf6jr4on0b5f5.apps.googleusercontent.com",
    ClientSecret: "lERovWunL6URp0GlG5bgL9cS",
    Scopes:       []string{"https://www.googleapis.com/auth/drive"},
    RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
    Endpoint:     google.Endpoint,
  }

  tok := new(oauth2.Token)

  usr, err := user.Current()
  if err != nil {
    return nil, err
  }
  var cachePath = usr.HomeDir + "/.haven-b-gdrivecreds"

  // try reading from the cache
  if cacheData, err := ioutil.ReadFile(cachePath); err == nil {
    err := json.Unmarshal(cacheData, &tok)
    if err != nil {
      log.Println("Found malformed data in cache file", cachePath)
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
    if err != nil {
      return nil, err
    }
    err = ioutil.WriteFile(cachePath, marshaled, 0400)
    if err != nil {
      return nil, err
    }
  }

  if tok.RefreshToken == "" {
    return nil, fmt.Errorf("failed to obtain a valid token")
  }

  client := config.Client(oauth2.NoContext, tok)

  return client, nil
}
