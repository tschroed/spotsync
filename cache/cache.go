package cache

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zmb3/spotify/v2"
)

const (
	searchesTable = "searches"
	searchesKey   = "query"
)

type Cache struct {
	db    *sql.DB
	debug bool
}

type Options struct {
	Debug bool
}

func New(filename string, o Options) (*Cache, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	return &Cache{
		db:    db,
		debug: o.Debug,
	}, nil
}

func (c *Cache) debugPrintln(v ...any) {
	if c.debug {
		fmt.Println(v)
	}
}

func (c *Cache) upsertAny(table string, key string, value any) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return err
	}
	s := buf.String()
	c.debugPrintln("encoding:", s)
	q := fmt.Sprintf("INSERT INTO %s VALUES(?,?,?);", table)
	res, err := c.db.Exec(q, key, time.Now(), s)
	if err != nil {
		return err
	}
	c.debugPrintln("res:", res)
	return nil
}

func (c *Cache) UpsertSearch(search string, result *spotify.SearchResult) error {
	return c.upsertAny(searchesTable, search, result)
}

func (c *Cache) lookupAny(table string, keyName string, key string, out any) error {
	var k, v string
	var t time.Time
	q := fmt.Sprintf("SELECT * FROM %s WHERE %s=?", table, keyName)
	row := c.db.QueryRow(q, key)
	if err := row.Scan(&k, &t, &v); err != nil {
		return err
	}
	c.debugPrintln("val:", v)
	buf := bytes.NewBufferString(v)
	err := json.NewDecoder(buf).Decode(out)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Search(search string) (*spotify.SearchResult, error) {
	var s spotify.SearchResult
	err := c.lookupAny(searchesTable, searchesKey, search, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
