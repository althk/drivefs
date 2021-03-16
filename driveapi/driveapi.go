// Package driveapi provides basic functionality to work with Google Drive API.
package driveapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func tokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("token-state", oauth2.AccessTypeOffline)
	fmt.Printf("Open the below link in your browser and "+
		"then type/paste the authorization code here:\n%v\n", authURL)

	var authzCode string
	if _, err := fmt.Scan(&authzCode); err != nil {
		log.Fatalf("Unable to read authorization code :%v", err)
	}
	tok, err := config.Exchange(context.TODO(), authzCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving token to file %v", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to save token to %v: %v\n", path, err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		log.Fatalf("Unable to write token to file %v: %v\n", path, err)
	}
}

func getClient(config *oauth2.Config, tokenPath string) *http.Client {

	tok, err := tokenFromFile(tokenPath)

	if err != nil {
		tok = tokenFromWeb(config)
		saveToken(tokenPath, tok)
	}
	return config.Client(context.Background(), tok)
}

const (
	MimeTypeGoogleDoc = iota
	MimeTypeGoogleDrawing
	MimeTypeGoogleDriveFile
	MimeTypeGoogleDriveFolder
	MimeTypeGoogleForm
	MimeTypeGoogleFusionTable
	MimeTypeGoogleMyMap
	MimeTypeGoogleSlide
	MimeTypeGoogleAppsScript
	MimeTypeShortcut
	MimeTypeGoogleSite
	MimeTypeGoogleSpreadsheet
)

// See https://developers.google.com/drive/api/v3/mime-types
var googleAppsMimeTypes = map[int]string{
	MimeTypeGoogleDoc:         "application/vnd.google-apps.document",
	MimeTypeGoogleDrawing:     "application/vnd.google-apps.drawing",
	MimeTypeGoogleDriveFile:   "application/vnd.google-apps.file",
	MimeTypeGoogleDriveFolder: "application/vnd.google-apps.folder",
	MimeTypeGoogleForm:        "application/vnd.google-apps.form",
	MimeTypeGoogleFusionTable: "application/vnd.google-apps.fusiontable",
	MimeTypeGoogleMyMap:       "application/vnd.google-apps.map",
	MimeTypeGoogleSlide:       "application/vnd.google-apps.presentation",
	MimeTypeGoogleAppsScript:  "application/vnd.google-apps.script",
	MimeTypeShortcut:          "application/vnd.google-apps.shortcut",
	MimeTypeGoogleSite:        "application/vnd.google-apps.site",
	MimeTypeGoogleSpreadsheet: "application/vnd.google-apps.spreadsheet",
}

func GoogleAppsMimeTypeText(code int) string {
	return googleAppsMimeTypes[code]
}

func InitWithConfigJSON(
	ctx context.Context, b []byte, tokenPath string) *drive.Service {
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse config from json: %v", err)
	}
	client := getClient(config, tokenPath)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Drive service: %v", err)
	}
	return service
}

func RootFolder(ctx context.Context, drv *drive.Service) File {
	root, err := drv.Files.Get("root").Context(ctx).
		Fields("id, name").Do()
	if err != nil {
		log.Fatalf("Error fetching root folder: %v", err)
		return nil
	}
	return &file{
		GD:       drv,
		id:       root.Id,
		name:     root.Name,
		files:    nil,
		mimeType: GoogleAppsMimeTypeText(MimeTypeGoogleDriveFolder),
	}
}

func (f *file) ListFiles(
	ctx context.Context) ([]File, error) {
	if !f.IsDir() {
		return nil, errors.New("not a directory")
	}
	log.Printf("Listing files for %s", f.Name())
	if time.Since(f.lsTime).Minutes() < 60 {
		return f.files, nil
	}
	var nextPageToken string
	var files []File
	for {
		res, err := f.GD.Files.List().Context(ctx).
			Fields("nextPageToken, files(id, name, size, parents, mimeType)").
			PageToken(nextPageToken).
			Q(fmt.Sprintf("'%s' in parents", f.id)).
			Do()
		if err != nil {
			return files, err
		}
		for _, e := range res.Files {
			files = append(files, &file{
				id:         e.Id,
				name:       e.Name,
				parentID:   f.ID(),
				parentName: f.Name(),
				size:       uint64(e.Size),
				mimeType:   e.MimeType,
				GD:         f.GD,
			})
		}
		if len(res.NextPageToken) == 0 {
			break
		}
		nextPageToken = res.NextPageToken
	}
	f.files = files
	f.lsTime = time.Now()
	return files, nil
}

type file struct {
	GD                                       *drive.Service
	id, name, mimeType, parentID, parentName string
	size                                     uint64
	content                                  []byte
	files                                    []File
	lsTime                                   time.Time
}

type File interface {
	String() string
	IsDir() bool
	IsGoogleAppsFile() bool
	ListFiles(ctx context.Context) ([]File, error)
	Size() uint64
	Name() string
	MimeType() string
	ParentID() string
	ParentName() string
	Content() []byte
	Files() []File
	ID() string
}

func (f *file) String() string {
	return fmt.Sprintf(
		"%s/%s => mime type: %s, ID: %s, size: %d KB",
		f.parentName, f.name, f.mimeType, f.id, f.size/1024)
}

func (f *file) IsDir() bool {
	return f.mimeType == GoogleAppsMimeTypeText(MimeTypeGoogleDriveFolder)
}

func (f *file) IsGoogleAppsFile() bool {
	return strings.HasPrefix(f.mimeType, "application/vnd.google-apps")
}

func (f *file) Size() uint64 {
	return f.size
}

func (f *file) Name() string {
	return f.name
}

func (f *file) ID() string {
	return f.id
}

func (f *file) MimeType() string {
	return f.mimeType
}

func (f *file) ParentName() string {
	return f.parentName
}

func (f *file) ParentID() string {
	return f.parentID
}

func (f *file) Content() []byte {
	return f.content
}

func (f *file) Files() []File {
	return f.files
}
