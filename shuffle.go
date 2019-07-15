package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/carrot/go-pinterest"
	"github.com/carrot/go-pinterest/controllers"
)

var staticAssetsDir = os.Getenv("STATIC_ASSETS_DIR")
var templatesDir = os.Getenv("TEMPLATES_DIR")
var tlsCertPath = os.Getenv("TLS_CERT_PATH")
var tlsKeyPath = os.Getenv("TLS_KEY_PATH")
var clientID = os.Getenv("CLIENT_ID")
var clientSecret = os.Getenv("CLIENT_SECRET")
var rootURL = os.Getenv("ROOT_URL")

var oauthURL = "https://api.pinterest.com/oauth/?response_type=code&redirect_uri=" + rootURL + "/redirect&client_id=" + clientID + "&scope=read_public"

var client *pinterest.Client

// Pin is the model data that we use in template HTML
type Pin struct {
	ImageURL string
	PinURL   string
	Color    string
}

// Board is the model data that we use in template HTML
type Board struct {
	Name     string
	URL      string
	PinCoint int32
}

// TemplateData is the main object we pass for templating HTML
type TemplateData struct {
	OAuthURL      string
	Authenticated bool
	Error         string
	Message       string
	Pins          []Pin
	Boards        []Board
}

// neuteredFileSystem is used to prevent directory listing of static assets
type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {

	log.Print(path)

	// Check if path exists
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	// If path exists, check if is a file or a directory.
	// If is a directory, stop here with an error saying that file
	// does not exist. So user will get a 404 error code for a file/directory
	// that does not exist, and for directories that exist.
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, os.ErrNotExist
	}

	// If file exists and the path is not a directory, let's return the file
	return f, nil
}

func redirectHandler(w http.ResponseWriter, req *http.Request) {
	codeKey := req.FormValue("code")

	if len(codeKey) > 0 {
		log.Println("Access Code: " + codeKey)

		accessToken, err := client.OAuth.Token.Create(clientID, clientSecret, codeKey)

		if err != nil {
			log.Println("Something went wrong with the redirect code")
			log.Println(err)

			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")

			io.WriteString(w, err.Error())
			return
		}

		client = client.RegisterAccessToken(accessToken.AccessToken)
		log.Println("Access Token: " + accessToken.AccessToken)

		cookie := http.Cookie{
			Name:    "access_token",
			Value:   accessToken.AccessToken,
			Expires: time.Now().Add(365 * 24 * time.Hour),
		}

		http.SetCookie(w, &cookie)

		log.Println("Success. Go to index!")

		http.Redirect(w, req, rootURL, http.StatusMovedPermanently)
	}
}

// renders page after passing some data to the HTML template
func indexHandler(w http.ResponseWriter, req *http.Request) {

	tmplData := TemplateData{
		OAuthURL:      oauthURL,
		Authenticated: false,
		Boards:        []Board{},
		Pins:          []Pin{},
	}

	accessTokenCookie, err := req.Cookie("access_token")

	if err == http.ErrNoCookie {
		log.Println("Missing Cookie")

	} else {
		log.Println("Cookie Exists")

		client = client.RegisterAccessToken(accessTokenCookie.Value)

		user, err := client.Me.Fetch()

		if err != nil {
			log.Println(err.Error())
			tmplData.Error = err.Error()

		} else {
			log.Println("User Authenticated")

			tmplData.Authenticated = true

			boards, err := client.Me.Boards.Fetch()

			if err != nil {
				log.Println(err.Error())
				tmplData.Error = err.Error()

			} else {
				for _, board := range *boards {
					userSlashBoard := strings.ToLower(user.Username + "/" + path.Base(board.Url))
					tmplData.Boards = append(tmplData.Boards, Board{
						Name:     board.Name,
						URL:      rootURL + "/?board=" + userSlashBoard,
						PinCoint: board.Counts.Pins,
					})
				}
			}

			boardKey := req.FormValue("board")

			if len(boardKey) > 0 {
				log.Println("Board Key: " + boardKey)

				pins, boardError := loadAllPinsFromBoard(boardKey, accessTokenCookie.Value)

				if boardError != nil {
					log.Println(boardError.Error())
					tmplData.Error = boardError.Error()

				} else {
					tmplData.Pins = *pins
				}
			} else {
				tmplData.Message = "You can select one of your boards in the bottom right. You can also use the URL: ?board=username/board"
			}
		}
	}

	// Build path to template
	tmplPath := filepath.Join(templatesDir, "layout.gohtml")
	// Load template from disk
	tmpl := template.Must(template.ParseFiles(tmplPath))

	tmpl.Execute(w, tmplData)
}

// returns a pointer to a slice containing ALL pins within a board
func loadAllPinsFromBoard(board string, accessToken string) (*[]Pin, error) {
	log.Println("Loading pins from: " + board)

	var allPins []Pin

	pins, cursor, err := loadPinsWithCursor(board, "")

	if err != nil {
		return nil, err
	}

	allPins = append(allPins, *pins...)

	for cursor != "" {
		pins, cursor, err = loadPinsWithCursor(board, cursor)
		allPins = append(allPins, *pins...)

		if err != nil {
			return nil, err
		}
	}

	return &allPins, nil
}

// returns a pointer to a slice containing 100 (max) pins on a page of a board
func loadPinsWithCursor(board string, cursor string) (*[]Pin, string, error) {
	var pinsOnPage []Pin

	pins, page, err := client.Boards.Pins.Fetch(
		board,
		&controllers.BoardsPinsFetchOptionals{
			Cursor: cursor,
			Limit:  100,
		},
	)

	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	for _, pin := range *pins {
		pinsOnPage = append(pinsOnPage, Pin{
			ImageURL: pin.Image.Original.Url,
			PinURL:   pin.Url,
			Color:    pin.Color,
		})
	}

	return &pinsOnPage, page.Cursor, nil
}

// httpsRedirect redirects http requests to https
func httpsRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(
		w, r,
		"https://"+r.Host+r.URL.String(),
		http.StatusMovedPermanently,
	)
}

func main() {
	client = pinterest.NewClient()

	// http to https redirection
	go http.ListenAndServe(":80", http.HandlerFunc(httpsRedirect))

	// Serve static files while preventing directory listing
	mux := http.NewServeMux()
	fs := http.FileServer(neuteredFileSystem{http.Dir(staticAssetsDir)})
	mux.Handle("/res/", http.StripPrefix("/res/", fs))

	log.Println(http.Dir(staticAssetsDir))

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/redirect", redirectHandler)

	// Launch TLS server
	log.Fatal(http.ListenAndServeTLS(":443", tlsCertPath, tlsKeyPath, mux))
}
