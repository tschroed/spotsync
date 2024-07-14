module github.com/tschroed/spotsync/v2

go 1.22.5

require (
	github.com/google/go-cmp v0.6.0
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/zmb3/spotify/v2 v2.4.2
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/tschroed/spotify/v2 v2.0.0-20240707013608-db8ad82919b6 // indirect
	github.com/tschroed/spotsync v0.0.0-00010101000000-000000000000 // indirect
	github.com/tschroed/spotsync/media v0.0.0-00010101000000-000000000000 // indirect
	github.com/zmb3/spotify v1.3.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/oauth2 v0.0.0-20210810183815-faf39c7919d5 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/tschroed/spotsync => /home/trevors/src/spotsync

replace github.com/tschroed/spotsync/media => /home/trevors/src/spotsync/media
