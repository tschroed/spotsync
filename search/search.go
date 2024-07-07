package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"log"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"

	"github.com/tschroed/spotsync"
)

var (
	dFlag = flag.Bool("d", false, "Enable debugging")
)

func main() {
	flag.Parse()
	text := strings.Join(os.Args[1:], " ")
	if text == "" {
		log.Fatal("Please supply search terms on the command line")
	}
	ctx := context.Background()
	o := authserver.Options{
		Debug:        *dFlag,
		Port:         8080,
		AuthPath:     "/callback",
		RedirectHost: "192.168.1.101",
		Scopes:       []string{spotifyauth.ScopeUserLibraryModify},
	}
	server := authserver.New(o)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
	url := server.AuthURL()
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client, err := server.Client()
	if err != nil {
		log.Fatal(err)
	}

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
