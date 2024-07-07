package media

import (
	"fmt"
	"iter"  // To use the iter package, export GOEXPERIMENT=rangefunc
	"log"
	"regexp"
	"os"
)

var (
	matcher = regexp.MustCompile("(?:[0-9]{2}\\.? )?(?P<name>.*)\\.(?:mp|MP)3")
)

func extractTrackName(filename string) string {
	if !matcher.MatchString(filename) {
		return filename
	}
	matches := matcher.FindStringSubmatch(filename)
	return matches[matcher.SubexpIndex("name")]
}

// AlbumMetadat contains metadata about an album, artist, name, tracks.
type AlbumMetadata struct {
	Artist string
	Name string
	Tracks []string
}

type AlbumIterFn iter.Seq[*AlbumMetadata]

// This may be entirely superfluous since it's just ranging over the channel
// at this point.
func AlbumIterator(ch chan *AlbumMetadata) AlbumIterFn {
	return func(yield func(*AlbumMetadata) bool) {
		for {
			a := <- ch
			if a == nil || !yield(a) {
				return
			}
		}
	}
}

type DirectoryReader func(name string) ([]os.DirEntry, error)

type directoryAlbumProducer struct {
	root string
	readDir DirectoryReader
	ch chan *AlbumMetadata
}

func (d *directoryAlbumProducer) Albums() AlbumIterFn {
	return AlbumIterator(d.ch)
}


func (d* directoryAlbumProducer) Start() {
	arts, err := d.readDir(d.root)
	l := log.New(os.Stderr, "dAP: ", log.Ldate|log.Ltime|log.Lshortfile)
	if err != nil {
		l.Println("warn:", err)
		close(d.ch)
		return
	}
	for _, art := range arts {
		if !art.IsDir() {
			l.Printf("warn: %s is not a directory", art.Name())
			continue
		}
		albs, err := d.readDir(fmt.Sprintf("%s/%s", d.root, art.Name()))
		if err != nil {
			l.Println("warn:", err)
			continue
		}
		for _, alb := range albs {
			if !alb.IsDir() {
				l.Printf("warn: %s is not a directory", alb.Name())
				continue
			}
			tracks, err := d.readDir(fmt.Sprintf("%s/%s/%s", d.root, art.Name(), alb.Name()))
			if err != nil {
				l.Println("warn:", err)
				continue
			}
			t := make([]string, 0)
			for _, track := range tracks {
				t = append(t, extractTrackName(track.Name()))
			}
			d.ch <- &AlbumMetadata{
				Artist: art.Name(),
				Name: alb.Name(),
				Tracks: t,
			}
		}
	}
	close(d.ch)
}

func NewDirectoryAlbumProducer(root string, readDir DirectoryReader) *directoryAlbumProducer {
	ch := make(chan *AlbumMetadata, 20)
	d := &directoryAlbumProducer{
		root: root,
		readDir: readDir,
		ch: ch,
	}
	return d
}

