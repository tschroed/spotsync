package authserver

import (
	"bytes"
	"fmt"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"io"
	"log"
	"net/http"

	"github.com/zmb3/spotify/v2"
)

type roundtripLogger struct {
	Transport http.RoundTripper
}

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

type Options struct {
	Debug        bool
	Port         uint16
	AuthPath     string
	RedirectHost string
	Scopes       []string
}

type AuthServer struct {
	opts  Options
	auth  *spotifyauth.Authenticator
	state string
	ch    chan *spotify.Client
}

func (s *AuthServer) completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := s.auth.Token(r.Context(), s.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != s.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, s.state)
	}
	// use the token to get an authenticated client
	httpClient := spotifyauth.New().Client(r.Context(), tok)
	if s.opts.Debug {
		httpClient.Transport = roundtripLogger{Transport: httpClient.Transport}
	}
	client := spotify.New(httpClient)
	fmt.Fprintf(w, "Login Completed!")
	s.ch <- client
}

// New creates a new authorization server. Start() must be called to start
// the server listening.
func New(opts Options) *AuthServer {
	s := &AuthServer{opts: opts}
	s.state = "lololol"
	s.ch = make(chan *spotify.Client)
	return s
}

// Start starts the authorization server in the background.
func (s *AuthServer) Start() error {
	redirectURI := fmt.Sprintf("http://%s:%d%s", s.opts.RedirectHost, s.opts.Port, s.opts.AuthPath)
	s.auth = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(s.opts.Scopes...))
	http.HandleFunc(s.opts.AuthPath, s.completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", s.opts.Port), nil)
		if err != nil {
			log.Fatal(err)
		}
	}()
	return nil
}

// AuthURL returns the Spotify authorization URL suitable for displaying to
// the user.
func (s *AuthServer) AuthURL() string {
	return s.auth.AuthURL(s.state)
}

// Client returns the spotify Client, blocking until it's available.
func (s *AuthServer) Client() (*spotify.Client, error) {
	return <-s.ch, nil
}
