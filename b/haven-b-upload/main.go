package main

import (
	"github.com/cenkalti/backoff"
	"github.com/docopt/docopt-go"
	"github.com/pivotal-golang/bytefmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"time"
	"bufio"
)

// use 16M chunk size .. ?
const PreferredChunkSize = 1024 * 1024 * 16

var client *http.Client

func main() {
	usage := `Upload a large stream from stdin to Google Drive.

Usage:
  haven-b-upload <human_filename> <parent_folder_id>

Options:
  -h --help     Show this screen.
  --version     Show version.`

	arguments, err := docopt.Parse(usage, nil, true, "0.1", false)
	name := arguments["<human_filename>"].(string)
	folder_id := arguments["<parent_folder_id>"].(string)
	if name == "" || folder_id == "" {
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


	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Fatalln("Refusing to read from terminal")
	}

	id := upload(os.Stdin, name, folder_id)
	fmt.Printf("%s\n", id)
}

/// Returns the Google Drive ID of the created file on success, panics on error.
/// `filename` may not contain JSON escape sequences (" and \).
func upload(in_raw io.Reader, filename, folder_id string) string {
	buffered := bufio.NewReaderSize(in_raw, 2*PreferredChunkSize)
	counter := io.LimitedReader{N: math.MaxInt64, R: buffered}
	hash := md5.New()
	in := io.TeeReader(&counter, hash)

	// I really want to do this properly, but parents is an array and I hate that.
	var metadataJson = []byte(fmt.Sprintf(`{"title":"%s", "parents":[{"kind": "drive#fileLink","id": "%s"}]}`, filename, folder_id))

	// Don't bother with retrying this first request; if the network is down, we can safely fail
	// and we probably have failed at the OAuth stage already.
	r, err := client.Post("https://www.googleapis.com/upload/drive/v2/files?uploadType=resumable",
		"application/json; charset=UTF-8",
		bytes.NewBuffer(metadataJson))
	if err != nil {
		log.Fatalln(err)
	}
	if r.StatusCode != http.StatusOK {
		log.Fatalln("failed to create upload")
	}
	uploadUrl := r.Header.Get("Location")
	log.Println("Metadata sent, starting upload...")

	var chunk bytes.Buffer
	chunk.Grow(PreferredChunkSize)

	// A word on sizes: the total sizes are int64 for .. you know, scalability.
	// However, buffer and chunk sizes are ints, as that's what golang uses for
	// memory management. Think twice if you feel like you have to cast
	// int64->int somewhere.
	// TODO: size_t is likely unsigned, should we settle for uint64 as well?

	// The current estimate for the total size of the input.
	// After the following loop terminates, it will be accurate.
	var total_size int64

	// The byte offset of the byte following the one that was last sent to the server.
	var chunkoffs int64

	eof := false

	for !eof {
		chunk.Reset()
		chunksize, err := io.Copy(&chunk, &io.LimitedReader{N: PreferredChunkSize, R: in})
		if err != nil {
			log.Fatalln(err)
		}
		total_size += chunksize
		if chunksize != PreferredChunkSize {
			// Short read; that means we've reached EOF (since err is nil).
			// Send this chunk, then we're done.
			eof = true
		}

		// only report the total size when finishing
		var reported_total_size int64
		if eof {
			reported_total_size = total_size
		}

		actual_range_md5 := uploadChunk(uploadUrl, chunk.Bytes(), chunkoffs, reported_total_size)

		s := fmt.Sprintf("(%s in)", bytefmt.ByteSize(uint64(chunkoffs+chunksize)))
		log.Println("Uploaded chunk", chunkoffs, "--", chunkoffs+chunksize, s)
		chunkoffs += chunksize

		if actual_range_md5 != "" {
			expected_range_md5 := hex.EncodeToString(hash.Sum(nil))
			if actual_range_md5 != expected_range_md5 {
				log.Fatalln("Range MD5 doesn't match: expected",expected_range_md5,"but got",actual_range_md5)
			}
		} else {
			// the last chunk doesn't carry a range md5
			if !eof {
				log.Fatalln("No range MD5 was returned, but not eof?")
			}
		}
	}

	log.Println("Reached end of input, finalizing..")

	// fetch the upload status one last time
	req, err := http.NewRequest("PUT", uploadUrl, nil)
	req.Header.Set("Content-Range", fmt.Sprintf("bytes */%d", total_size))
	resp, err := doWithRetry(req, []byte{}, []int{200})
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode != 200 {
		x, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		log.Fatalln("failed to finalize upload", string(x))
	}
	outputMetadataJson, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var meta GdriveFileMeta
	err = json.Unmarshal(outputMetadataJson, &meta)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("OK, Google has all", total_size, "bytes")

	// Sanity check 1:
	// Make sure we have actually drained the input.
	read := math.MaxInt64 - counter.N
	if read != total_size {
		log.Fatalln("whoops -- uploaded", total_size, "bytes, but read", read)
	}
	var tmpbuf bytes.Buffer
	tmpbuf.Grow(10)
	n, _ := in_raw.Read(tmpbuf.Bytes())
	if n != 0 {
		log.Fatalln("didn't drain input buffer -- read", n, "instead of 0")
	}

	// Sanity check 2:
	// Check the md5sum of the gdrive file against what Google tells us
	expected_md5 := hex.EncodeToString(hash.Sum(nil))
	actual_md5 := meta.Md5

	if expected_md5 != actual_md5 {
		log.Fatalln("final md5 check failed: expected", expected_md5, "but Google has", actual_md5)
	} else {
		log.Println("OK, MD5 check passed --", expected_md5)
	}

	if meta.Id == "" {
		log.Fatalln("file doesn't have an ID!")
	}

	if meta.Size != strconv.FormatInt(total_size, 10) {
		log.Fatalln("reported file size is", meta.Size, " -- but expected to have", total_size)
	}

	return meta.Id
}

type GdriveFileMeta struct {
	Md5  string `json:"md5Checksum"`
	Size string `json:"fileSize"`
	Id   string `json:"id"`
}

// Our backoff parameters, chosen to allow grotesquely long intervals.
// It's not like anyone else cares (or should care!) -- if you pipe `zfs send` or `tar`
// into this, they can wait as long as needed.
var timer = &backoff.ExponentialBackOff{
	InitialInterval:     500 * time.Millisecond,
	RandomizationFactor: 0.5,
	Multiplier:          1.5,
	MaxInterval:         1 * time.Minute,
	MaxElapsedTime:      30 * time.Hour, // Google deletes the temp state after 1d, so it doesn't make much sense to wait longer
	Clock:               backoff.SystemClock,
}

func uploadChunk(uploadUrl string, chunk []byte, offset int64, reported_total_size int64) string {
	reported_total_size_fmt := "*"
	if reported_total_size != 0 {
		reported_total_size_fmt = fmt.Sprintf("%d", reported_total_size)
	}
	done_in_chunk := 0
	chunksize := len(chunk)

	range_md5 := ""

	for done_in_chunk < chunksize {
		nextoffs := offset + int64(done_in_chunk)
		req, err := http.NewRequest("PUT", uploadUrl, nil)
		if err != nil {
			log.Fatalln(err)
		}

		req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%s", nextoffs, offset+int64(chunksize-1), reported_total_size_fmt))
		resp, err := doWithRetry(req, chunk[done_in_chunk:], []int{308, 201, 200, 503})
		if err != nil {
			log.Fatalln(err)
		}
		if resp.StatusCode == 201 || resp.StatusCode == 200 {
			log.Println("(uploaded the final chunk -- not checking ranges)")
			// the final chunk doesn't even include the range -- so we can't check it here.
			// let's just assume everything has gone right..?
			// Also, don't return an x-range value.
			return ""
		} else if resp.StatusCode == 503 {
			log.Println("Upload was interrupted... let's see if we can recover")
			// do a lookup request to see where we're at
			lookup_req, err := http.NewRequest("PUT", uploadUrl, nil)
			lookup_req.Header.Set("Content-Range", fmt.Sprintf("bytes */%s", reported_total_size_fmt))
			resp, err = doWithRetry(lookup_req, []byte{}, []int{308, 201, 200})
			if err != nil {
				log.Fatalln(err)
			}

			// we don't look at `resp` here, but the range calculation code below does
		} else {
			// it's not the last chunk, so bail if Google doesn't tell about incomplete updates
			if resp.StatusCode != 308 {
				log.Fatalln("failed to upload chunk")
			}
		}

		// At the end, we'll return the most recent range md5
		range_md5 = resp.Header.Get("X-Range-Md5")

		// Check the range google reports and update our current state
		var reported_start int64
		var reported_end int64
		n, err := fmt.Sscanf(resp.Header.Get("Range"), "bytes=%d-%d", &reported_start, &reported_end)
		if err != nil || n != 2 {
			log.Fatalln("failed to parse Range header", resp.Header.Get("Range"))
		}
		if reported_start != 0 {
			log.Fatalln("insane reported_start, expected", 0, "got", reported_start)
		}
		end_of_chunk := offset+int64(chunksize-1)

		// Calculate how many bytes from the chunk didn't get uploaded yet.
		// This can happen when the upload gets interrupted... sigh.
		// The `int` cast is safe since chunksize < MAX_INT.
		bytes_missing_from_chunk := int(end_of_chunk - reported_end)
		if bytes_missing_from_chunk < 0 {
			log.Fatalln("insane reported_end; chunk_end is at", end_of_chunk, "but Google says it has until", reported_end)
		}

		if bytes_missing_from_chunk > 0 {
			log.Println("Chunk was partially uploaded (missing",bytes_missing_from_chunk,"bytes), recovering..")
		}

		done_in_chunk = chunksize-bytes_missing_from_chunk

		// Add 1 since done_in_chunk points to the next unsent byte, while reported_end
		// points to the last sent one.
		if int64(done_in_chunk) != (reported_end+1)-offset {
			log.Fatalln("failed to calculate chunk boundary; expected to be done with",done_in_chunk,"but Google says we're at",reported_end+1-offset)
		}

		if done_in_chunk < chunksize {
			log.Println("... done with",done_in_chunk,"out of the",chunksize,"bytes in the chunk")
		}
	}

	return range_md5
}

func doWithRetry(req *http.Request, body []byte, allowedStatuses []int) (*http.Response, error) {
	var resp *http.Response
	i := 0
	err := backoff.Retry(func() error {
		i++
		log.Println("Trying:",req,"with",len(body))

		// Create a new reader every time so that we correctly rewind to the
		// beginning of the buffer on every retry.
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))

		var int_err error
		resp, int_err = client.Do(req)
		log.Println("Got",resp)
		if int_err == nil {
			ok := false
			for _, i := range allowedStatuses {
				if resp.StatusCode == i {
					ok = true
					break
				}
			}
			if !ok {
				int_err = fmt.Errorf("got status %d, expected within %x", resp.StatusCode, allowedStatuses)
			}
		}
		if int_err != nil {
			log.Println("Attempt", i, "failed:", int_err)
			log.Println(req)
			log.Println(resp)
			if resp != nil {
				slurp, serr := ioutil.ReadAll(resp.Body)
				if serr != nil {
					log.Println("Also, failed to slurp response body")
				} else {
					log.Println(string(slurp))
				}
			}
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
