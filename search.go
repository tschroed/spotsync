package main

import (
	"bytes"
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"
)

type roundtripLogger struct {
	Transport http.RoundTripper
}

var (
	dFlag = flag.Bool("d", false, "Enable debugging")
	auth  = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserLibraryModify))
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func (r roundtripLogger) RoundTrip(request *http.Request) (*http.Response, error) {
	fmt.Println("request:", request.Method, request.URL.String())
	fmt.Println("headers:")
	for k, v := range request.Header {
		fmt.Printf("  %s: %s", k, v)
	}
	res, err := r.Transport.RoundTrip(request)
	var bufReader bytes.Buffer
	io.Copy(&bufReader, res.Body)
	fmt.Println("response", bufReader.String())
	// res.Body is already closed. you need to make a copy again to pass it for the code
	res.Body = io.NopCloser(bytes.NewReader(bufReader.Bytes()))
	return res, err
}


// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://192.168.1.101:8080/callback"

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	httpClient := spotifyauth.New().Client(r.Context(), tok)
	if (*dFlag) {
		httpClient.Transport = roundtripLogger{Transport: httpClient.Transport}
	}
	client := spotify.New(httpClient)
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}

func main() {
	flag.Parse()
	text := strings.Join(os.Args[1:], " ")
	if text == "" {
		log.Fatal("Please supply search terms on the command line")
	}
	ctx := context.Background()
	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)

	results, err := client.Search(ctx, text, spotify.SearchTypeArtist|spotify.SearchTypeAlbum)
	if err != nil {
		log.Fatal(err)
	}

	// handle album results
	if results.Albums != nil {
	    reader := bufio.NewReader(os.Stdin)
		fmt.Println("Albums:")
		toAdd := make([]spotify.SimpleAlbum, 0)
		for _, item := range results.Albums.Albums {
			fmt.Println("   ", item.Name)
			fmt.Println("    >> Artists:")
			for _, artist := range item.Artists {
				fmt.Println("        ", artist.Name)
			}
			fa, err := client.GetAlbum(ctx, item.ID)
			if err != nil {
				log.Print(err)
			}
			fmt.Println("    >> Tracks:")
			for _, track := range fa.Tracks.Tracks { // Assume just 1 page
				fmt.Println("        ", track.Name)
			}
			fmt.Print("Add to library? [y/N] => ")
			r, _ := reader.ReadString('\n')
			r = strings.TrimSpace(r)
			if r == "y" || r == "Y" {
			  toAdd = append(toAdd, item)
			}
		}
		if len(toAdd) > 0 {
			fmt.Println("Adding...")
			ids := make([]spotify.ID, len(toAdd))
			for i, alb := range toAdd {
				fmt.Println("    ", alb.Artists[0].Name, " / ", alb.Name)
				ids[i] = alb.ID
			}
			err = client.AddAlbumsToLibrary(ctx, ids...)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
