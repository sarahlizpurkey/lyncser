package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	// Path where OAuth client credentials are stored
	credentialsFilePath = "~/.config/lyncser/credentials.json"
	// Path where the OAuth token will be stored
	tokenFilePath = "~/.config/lyncser/token.json"
)

// getClient retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := realPath(tokenFilePath)
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// getTokenFromWeb requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// tokenFromFile retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// getService returns a service that can be used to make API calls
func getService() *drive.Service {
	b, err := ioutil.ReadFile(realPath(credentialsFilePath))
	checkError(err)

	// If modifying these scopes, delete your previously saved token.json.
	clientConfig, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	checkError(err)
	client := getClient(clientConfig)

	service, err := drive.New(client)
	checkError(err)
	return service
}

// getFileList gets the list of file that this app has access to.
func getFileList(service *drive.Service) []*drive.File {
	listFilesCall := service.Files.List()
	listFilesCall.Fields("files(name, id, parents, modifiedTime), nextPageToken")
	listFilesCall.Q("trashed=false")
	var files []*drive.File
	for true {
		driveFileList, err := listFilesCall.Do()
		checkError(err)
		files = append(files, driveFileList.Files...)
		if driveFileList.NextPageToken == "" {
			break
		}
		listFilesCall.PageToken(driveFileList.NextPageToken)
	}
	return files
}

// createDir creates a directory in Google Drive. Also creates any necessary parent directories.
// Returns the Id of the directory created.
func createDir(service *drive.Service, name string, mapPaths map[string]string, lyncserRoot string) string {
	if name == "" || name == "." || name == "/" {
		return lyncserRoot
	}
	dirId, ok := mapPaths[name]
	if ok {
		return dirId // This directory already exists
	}
	parent := filepath.Dir(name)
	parentId, ok := mapPaths[parent]
	if !ok {
		// The parent directory does not exist either. Recursively create it.
		parentId = createDir(service, parent, mapPaths, lyncserRoot)
	}
	d := &drive.File{
		Name:     filepath.Base(name),
		MimeType: "application/vnd.google-apps.folder",
	}
	if parentId != "" {
		d.Parents = []string{parentId}
	}

	file, err := service.Files.Create(d).Do()
	checkError(err)
	fmt.Printf("Directory '%s' successfully created\n", name)
	mapPaths[name] = file.Id
	return file.Id
}

// createFile creates the file in Google Drive.
func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).Do()
	checkError(err)
	return file, nil
}
