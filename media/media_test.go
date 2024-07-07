package media

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAlbumIterator(t *testing.T) {
	want := []AlbumMetadata{
		AlbumMetadata{
			Artist: "Artist1",
			Name:   "Title1",
			Tracks: []string{
				"Track1",
				"Track2",
			},
		},
		AlbumMetadata{
			Artist: "Artist2",
			Name:   "Title2",
			Tracks: []string{
				"Track3",
				"Track4",
			},
		},
	}
	ch := make(chan *AlbumMetadata, len(want))
	for _, a := range want {
		ch <- &a
	}
	close(ch)
	got := make([]AlbumMetadata, len(want))
	i := 0
	for a := range AlbumIterator(ch) {
		got[i] = *a
		i++
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AlbumIterator mismatch (-want +got):\n%s", diff)
	}
}

func TestDirectoryAlbumIterator(t *testing.T) {
	want := []AlbumMetadata{
		AlbumMetadata{
			Artist: "Artist1",
			Name:   "Title1",
			Tracks: []string{
				"Track1",
				"Track2",
			},
		},
		AlbumMetadata{
			Artist: "Artist2 Two",
			Name:   "Title2 Two",
			Tracks: []string{
				"Track3 Three",
				"Track4",
			},
		},
		AlbumMetadata{
			Artist: "Artist2 Two",
			Name:   "Title3Three",
			Tracks: []string{
				"Track5 Five",
				"Track6 6",
			},
		},
	}
	prefixes := []string{
		"",
		"01 ",
		"02. ",
	}
	suffixes := []string{
		".mp3",
		".MP3",
	}
	tmp := t.TempDir()
	i := 0
	for _, alb := range want {
		if err := os.MkdirAll(fmt.Sprintf("%s/%s/%s", tmp, alb.Artist, alb.Name), 0750); err != nil {
			t.Error(err)
			continue
		}
		for _, track := range alb.Tracks {
			if _, err := os.Create(fmt.Sprintf("%s/%s/%s/%s%s%s", tmp, alb.Artist, alb.Name, prefixes[i%len(prefixes)], track, suffixes[i%len(suffixes)])); err != nil {
				t.Error(err)
				continue
			}
			i++
		}
	}

	d := NewDirectoryAlbumProducer(tmp, os.ReadDir)
	go func() {
		d.Start()
	}()
	got := make([]AlbumMetadata, len(want))
	i = 0
	for a := range d.Albums() {
		got[i] = *a
		i++
	}
	// Because Track1 ends up "Track1.mp3" and Track2 ends up "01.
	// Track2.MP3" they sort differently. Swap them.
	want[0].Tracks[0], want[0].Tracks[1] = want[0].Tracks[1], want[0].Tracks[0]
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Album mismatch (-want +got):\n%s", diff)
	}
}
