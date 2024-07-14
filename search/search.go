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
	"github.com/tschroed/spotsync/authserver"
	"github.com/tschroed/spotsync/cache"
	"github.com/tschroed/spotsync/media"
)

const (
	searchType = spotify.SearchTypeArtist | spotify.SearchTypeAlbum
)

const (
	MATCH_UNKNOWN    = iota
	MATCH_EXACT      = iota
	MATCH_SRC_PREFIX = iota
	MATCH_DST_PREFIX = iota
)

var (
	cFlag = flag.String("c", "/home/trevors/spotify.db", "Spotify cache sqlite database")
	dFlag = flag.Bool("d", false, "Enable debugging")
	lFlag = flag.String("l", "/usr/local/mp3", "Location of mp3 library")
)

func debug(format string, v ...any) {
	if *dFlag {
		log.Printf(format, v...)
	}
}

func bestMatch(artistName string, albumName string, albums []spotify.SimpleAlbum) (*spotify.SimpleAlbum, int) {
	art := spotsync.CanonicalizeName(artistName)
	alb := spotsync.CanonicalizeName(albumName)
	for _, al := range albums {
		cn := spotsync.CanonicalizeName(al.Name)
		debug("al cn: %s\n", cn)
		if cn != alb {
			debug("%s is not %s\n", alb, cn)
			continue
		}
		for _, ar := range al.Artists {
			cn = spotsync.CanonicalizeName(ar.Name)
			debug("ar cn: %s\n", cn)
			if cn == art {
				log.Printf("%s / %s seems to be an exact match\n", ar.Name, al.Name)
				return &al, MATCH_EXACT
			}
		}
	}
	for _, al := range albums {
		cn := spotsync.CanonicalizeName(al.Name)
		debug("al cn: %s\n", cn)
		if !strings.HasPrefix(cn, alb) {
			debug("%s is not %s\n", alb, cn)
			continue
		}
		for _, ar := range al.Artists {
			cn = spotsync.CanonicalizeName(ar.Name)
			debug("ar cn: %s\n", cn)
			if strings.HasPrefix(cn, art) {
				log.Printf("%s / %s seems to be a match\n", ar.Name, al.Name)
				return &al, MATCH_SRC_PREFIX
			}
		}
	}
	for _, al := range albums {
		cn := spotsync.CanonicalizeName(al.Name)
		debug("al cn: %s\n", cn)
		if !strings.HasPrefix(alb, cn) {
			debug("%s is not %s\n", alb, cn)
			continue
		}
		for _, ar := range al.Artists {
			cn = spotsync.CanonicalizeName(ar.Name)
			debug("ar cn: %s\n", cn)
			if strings.HasPrefix(cn, art) {
				log.Printf("%s / %s seems to be a match\n", ar.Name, al.Name)
				return &al, MATCH_DST_PREFIX
			}
		}
	}
	return nil, MATCH_UNKNOWN
}

func main() {
	flag.Parse()
	m := media.NewDirectoryAlbumProducer(*lFlag, os.ReadDir)
	go func() {
		m.Start()
	}()
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
		Scopes:       []string{spotifyauth.ScopeUserLibraryRead, spotifyauth.ScopeUserLibraryModify},
	}
	server := authserver.New(o)
	go func() {
		err := server.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

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

	c, err := cache.New(*cFlag, cache.Options{Debug: *dFlag})
	if err != nil {
		panic(err)
	}

	for alb := range m.Albums() {
		artName := strings.TrimPrefix(alb.Artist, "The ")
		albName := strings.TrimPrefix(alb.Name, "The ")
//		text := fmt.Sprintf("artist:\"%s\" album:\"%s\"", artName, albName)
		text := fmt.Sprintf("%s %s", artName, albName)
		fmt.Println(">> Searching for", text)

		// TODO: this should be refactored into e.g. SearchWithCache.
		results, err := c.Search(text)
		if err != nil {
			log.Println("[warn] Cache search failed:", err)
		}
		if results == nil {
			log.Println("[info] Searching Spotify")
			results, err = client.Search(ctx, text, searchType)
			if err != nil {
				log.Fatal(err)
			}
			if err := c.UpsertSearch(text, results); err != nil {
				log.Println("[warn] Failed to upsert search into cache:", err)
			}
		} else {
			log.Println("[info] Found results in cache")
		}

		// handle album results
		if results.Albums == nil || len(results.Albums.Albums) == 0 {
			fmt.Println("!! Failed to find", text)
			continue
		}
		reader := bufio.NewReader(os.Stdin)
		toAdd := make([]spotify.SimpleAlbum, 0)
		albums := results.Albums.Albums
		res, match := bestMatch(artName, albName, albums)
		if res == nil {
			log.Println("[warn] Found no good match.")
		} else {
			albums = []spotify.SimpleAlbum{*res}
		}
		fmt.Println("Albums:")
		for _, item := range albums {

			fmt.Println("   ", item.Name)
			fmt.Println("    >> Artists:")
			for _, artist := range item.Artists {
				fmt.Println("        ", artist.Name)
			}
			has, err := client.UserHasAlbums(ctx, item.ID)
			if err != nil {
				fmt.Println("err:", err)
				continue
			}
			if has[0] {
				fmt.Println("user already has ", item.Artists[0].Name, "/", item.Name, "considered a match")
				break
			}
			if match == MATCH_EXACT {
				toAdd = append(toAdd, item)
			} else {
				log.Println("[info] Match was not exact, so prompting...")
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
					break
				}
			}
		}
		if len(toAdd) > 0 {
			fmt.Println("Adding...")
			ids := make([]spotify.ID, len(toAdd))
			for i, alb := range toAdd {
				fmt.Println("    ", alb.Artists[0].Name, " / ", albName)
				ids[i] = alb.ID
			}
			err = client.AddAlbumsToLibrary(ctx, ids...)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
