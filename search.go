package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/zmb3/spotify/v2"
)

type roundtripLogger struct {
	Transport http.RoundTripper
}

var dFlag = flag.Bool("d", false, "Enable debugging")

func (r roundtripLogger) RoundTrip(request *http.Request) (*http.Response, error) {
	fmt.Println("request", request.URL.String())
	res, err := r.Transport.RoundTrip(request)
	var bufReader bytes.Buffer
	io.Copy(&bufReader, res.Body)
	fmt.Println("response", bufReader.String())
	// res.Body is already closed. you need to make a copy again to pass it for the code
	res.Body = io.NopCloser(bytes.NewReader(bufReader.Bytes()))
	return res, err
}

func main() {
	flag.Parse()
	text := strings.Join(os.Args[1:], " ")
	if text == "" {
		log.Fatal("Please supply search terms on the command line")
	}
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		log.Fatalf("couldn't get token: %v", err)
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	if (*dFlag) {
		httpClient.Transport = roundtripLogger{Transport: httpClient.Transport}
	}
	client := spotify.New(httpClient)
	results, err := client.Search(ctx, text, spotify.SearchTypeArtist|spotify.SearchTypeAlbum)
	if err != nil {
		log.Fatal(err)
	}

	// handle album results
	if results.Albums != nil {
		fmt.Println("Albums:")
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
		}
	}
}
