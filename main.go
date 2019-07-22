package main

import (
	"log"
	"net/http"
	"os"

	"github.com/carrot/go-pinterest"
)

var staticAssetsDir = os.Getenv("STATIC_ASSETS_DIR")
var templatesDir = os.Getenv("TEMPLATES_DIR")
var tlsCertPath = os.Getenv("TLS_CERT_PATH")
var tlsKeyPath = os.Getenv("TLS_KEY_PATH")
var clientID = os.Getenv("CLIENT_ID")
var clientSecret = os.Getenv("CLIENT_SECRET")
var rootURL = os.Getenv("ROOT_URL")

var client *pinterest.Client

func main() {
	// http to https redirection
	go http.ListenAndServe(":80", http.HandlerFunc(httpsRedirect))

	client = pinterest.NewClient()

	fs := http.FileServer(
		neuteredFileSystem{http.Dir(staticAssetsDir)},
	)

	mux := http.NewServeMux()
	mux.Handle("/res/", http.StripPrefix("/res/", fs))
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/redirect", pinterestRedirectHandler)

	// Launch TLS server
	log.Fatal(http.ListenAndServeTLS(":443", tlsCertPath, tlsKeyPath, mux))
}
