package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/go-oauth/oauth"
	"github.com/joeshaw/envdecode"
)

func startTwitterStream(stopchan <-chan struct{}, votes chan<- string) <-chan struct{} {
	stoppedchan := make(chan struct{}, 1)
	go func() {
		defer func() {
			stoppedchan <- struct{}{}
		}()
		for {
			select {
			case <-stopchan:
				log.Println("stopping Twitter...")
				return // return and trigger defered stop event
			default:
				log.Println("Querying Twitter...")
				// readFromTwiiter will block until it disconnects
				readFromTwitter(votes)
				log.Println("  (wating)")
				time.Sleep(10 * time.Second) // wait before reconnecting
			}
		}
	}()
	// We return a receive only chan though internally we can still send to it
	return stoppedchan
}

type tweet struct {
	Text string
}

// readFromTwitter connects to the twiiter streaming api, listens for
// configured votes and sends them on the votes channel.
func readFromTwitter(votes chan<- string) {
	options, err := loadOptions()
	if err != nil {
		log.Println("failed to load options:", err)
		return
	}
	u, err := url.Parse("https://stream.twitter.com/1.1/statuses/filter.json")
	if err != nil {
		log.Println("creating filter request failed:", err)
		return
	}
	query := make(url.Values)
	query.Set("track", strings.Join(options, ","))
	req, err := http.NewRequest("POST", u.String(), strings.NewReader(query.Encode()))
	if err != nil {
		log.Println("creating filter request failed:", err)
		return
	}
	// set auth headers and send requet
	resp, err := makeRequest(req, query)
	if err != nil {
		log.Println("making request failed", err)
		return
	}
	reader := resp.Body
	decoder := json.NewDecoder(reader)
	for {
		var t tweet
		// Internally, reader will block until something is sent.
		// Decode() will then parse the response and return
		if err := decoder.Decode(&t); err != nil {
			// the connection was probably closed
			break
		}
		for _, option := range options {
			if strings.Contains(
				strings.ToLower(t.Text),
				strings.ToLower(option),
			) {
				log.Println("vote (", option, "):", t.Text)
				votes <- option
			}
		}
	}
}

var (
	authSetupOnce sync.Once
	httpClient    *http.Client
)

// makeRequest sets auth headers and sends request to twitter api
func makeRequest(req *http.Request, params url.Values) (*http.Response, error) {
	authSetupOnce.Do(func() {
		setupTwitterAuth()
		httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: dial,
			},
		}
	})
	formEnc := params.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))
	req.Header.Set("Authorization", authClient.AuthorizationHeader(creds,
		"POST",
		req.URL,
		params))
	return httpClient.Do(req)
}

var conn net.Conn

// Connect to twitter
func dial(netw, add string) (net.Conn, error) {
	if conn != nil {
		conn.Close()
		conn = nil
	}
	netc, err := net.DialTimeout(netw, add, 5*time.Second)
	if err != nil {
		return nil, err
	}
	conn = netc
	return netc, nil
}

var reader io.ReadCloser

func closeConn() {
	if conn != nil {
		conn.Close()
	}
	if reader != nil {
		reader.Close()
	}
}

var (
	authClient *oauth.Client
	creds      *oauth.Credentials
)

func setupTwitterAuth() {
	var ts struct {
		ConsumerKey    string `env:"TWITTER_KEY,required"`
		ConsumerSecret string `env:"TWITTER_SECRET,required"`
		AccessToken    string `env:"TWITTER_ACCESS_TOKEN,required"`
		AccessSecret   string `env:"TWITTER_ACCESS_SECRET,required"`
	}
	if err := envdecode.Decode(&ts); err != nil {
		log.Fatal(err)
	}
	creds = &oauth.Credentials{
		Token:  ts.AccessToken,
		Secret: ts.AccessSecret,
	}
	authClient = &oauth.Client{
		Credentials: oauth.Credentials{
			Token:  ts.ConsumerKey,
			Secret: ts.ConsumerSecret,
		},
	}
}
