package cache

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zmb3/spotify/v2"
)

const (
	testDBFile = "test.db"
	schemaFile = "cache.sql"
)

func initCache(fname string) (*Cache, error) {
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		return nil, err
	}
	c, err := New(fname, Options{})
	if err != nil {
		return nil, err
	}
	_, err = c.db.Exec(string(b))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func customComparers() []cmp.Option {
	return []cmp.Option{
		cmp.Comparer(func(x, y spotify.SimpleAlbumPage) bool {
			return cmp.Equal(x.Albums, y.Albums)
		}),
	}
}

func TestAlbumInsert(t *testing.T) {
	const query = "foo"
	const never = "bar"
	fname := fmt.Sprintf("%s/%s", t.TempDir(), testDBFile)
	c, err := initCache(fname)
	if err != nil {
		t.Fatalf("initCache(\"%s\"): %v", fname, err)
	}
	r, err := c.Search(query)
	// It's expected to return an empty result set, which is promoted to error.
	if err == nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", query, r, err)
	}
	if r != nil {
		t.Errorf("c.Search(\"%s\") wasn't nil: %v", query, r)
	}

	r = &spotify.SearchResult{
		Albums: &spotify.SimpleAlbumPage{
			Albums: []spotify.SimpleAlbum{
				{
					Name: "SimpleName1",
					ID:   "album1",
					Artists: []spotify.SimpleArtist{
						{
							Name: "SimpleArtist",
							ID:   "artist1",
						},
					},
				},
				{
					Name: "SimpleName2",
					ID:   "album2",
					Artists: []spotify.SimpleArtist{
						{
							Name: "SimpleArtist",
							ID:   "artist2",
						},
					},
				},
			},
		},
	}
	err = c.UpsertSearch(query, r)
	if err != nil {
		t.Errorf("c.UpsertSearch(\"%s\", ...): %v", query, err)
	}

	got, err := c.Search(query)
	if err != nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", query, r, err)
	}
	diff := cmp.Diff(*r, *got, customComparers()...)
	if diff != "" {
		t.Errorf("c.Search(\"%s\") -got, +wanted: %s", query, diff)
	}
	c.Close()

	c, err = New(fname, Options{})
	if err != nil {
		t.Fatalf("New(\"%s\"): %v", fname, err)
	}
	got, err = c.Search(query)
	if err != nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", query, r, err)
	}
	diff = cmp.Diff(*r, *got, customComparers()...)
	if diff != "" {
		t.Errorf("c.Search(\"%s\") -got, +wanted: %s", query, diff)
	}
	r, err = c.Search(never)
	// It's expected to return an empty result set, which is promoted to error.
	if err == nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", never, r, err)
	}
	if r != nil {
		t.Errorf("c.Search(\"%s\") wasn't nil: %v", never, r)
	}
	c.Close()
}

func TestAlbumUpdate(t *testing.T) {
	const query = "foo"
	fname := fmt.Sprintf("%s/%s", t.TempDir(), testDBFile)
	fmt.Println("fname:", fname)
	c, err := initCache(fname)
	if err != nil {
		t.Fatalf("initCache(\"%s\"): %v", fname, err)
	}
	defer c.Close()
	r, err := c.Search(query)
	// It's expected to return an empty result set, which is promoted to error.
	if err == nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", query, r, err)
	}
	if r != nil {
		t.Errorf("c.Search(\"%s\") wasn't nil: %v", query, r)
	}

	r1 := &spotify.SearchResult{
		Albums: &spotify.SimpleAlbumPage{
			Albums: []spotify.SimpleAlbum{
				{
					Name: "SimpleName1",
					ID:   "album1",
					Artists: []spotify.SimpleArtist{
						{
							Name: "SimpleArtist",
							ID:   "artist1",
						},
					},
				},
				{
					Name: "SimpleName2",
					ID:   "album2",
					Artists: []spotify.SimpleArtist{
						{
							Name: "SimpleArtist",
							ID:   "artist2",
						},
					},
				},
			},
		},
	}
	err = c.UpsertSearch(query, r1)
	if err != nil {
		t.Errorf("c.UpsertSearch(\"%s\", ...): %v", query, err)
	}

	got, err := c.Search(query)
	if err != nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", query, r1, err)
	}
	diff := cmp.Diff(*r1, *got, customComparers()...)
	if diff != "" {
		t.Errorf("c.Search(\"%s\") -got, +wanted: %s", query, diff)
	}

	r2 := &spotify.SearchResult{
		Albums: &spotify.SimpleAlbumPage{
			Albums: []spotify.SimpleAlbum{
				{
					Name: "SimpleName3",
					ID:   "album3",
					Artists: []spotify.SimpleArtist{
						{
							Name: "SimpleArtist",
							ID:   "artist3",
						},
					},
				},
				{
					Name: "SimpleName4",
					ID:   "album4",
					Artists: []spotify.SimpleArtist{
						{
							Name: "SimpleArtist",
							ID:   "artist4",
						},
					},
				},
			},
		},
	}
	err = c.UpsertSearch(query, r2)
	if err != nil {
		t.Errorf("c.UpsertSearch(\"%s\", ...): %v", query, err)
	}

	got, err = c.Search(query)
	if err != nil {
		t.Errorf("c.Search(\"%s\"): %v, %v", query, r2, err)
	}
	diff = cmp.Diff(*r2, *got, customComparers()...)
	if diff != "" {
		t.Errorf("c.Search(\"%s\") -got, +wanted: %s", query, diff)
	}

	q := fmt.Sprintf("SELECT * FROM %s", searchesTable)
	rows, err := c.db.Query(q)
	if err != nil {
		t.Fatalf("c.db.Query(\"%s\"): %v", q, err)
	}
	i := 0
	for rows.Next() {
		i++
	}
	if i != 1 {
		t.Errorf("Row count mismatch. Got %d, wanted 1", i)
	}
}
