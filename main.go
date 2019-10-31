package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"

	pinterest "github.com/a-frony/go-pinterest"
	"golang.org/x/crypto/acme/autocert"
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
	// go http.ListenAndServe(":80", http.HandlerFunc(httpsRedirect))

	client = pinterest.NewClient()

	fs := http.FileServer(
		neuteredFileSystem{http.Dir(staticAssetsDir)},
	)

	log.Println(http.Dir(staticAssetsDir))

	mux := http.NewServeMux()
	mux.Handle("/res/", http.StripPrefix("/res/", fs))
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/redirect", pinterestRedirectHandler)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("shuffle.iggyzuk.com"),                   //Your domain here
		Cache:      autocert.DirCache("/etc/letsencrypt/live/shuffle.iggyzuk.com/"), //Folder for storing certificates
	}

	server := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: mux,
	}

	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

	// Launch TLS server
	log.Fatal(server.ListenAndServeTLS("", ""))
}
