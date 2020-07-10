package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"receiptstracker-api/external"
	"receiptstracker-api/httpserver"
	"receiptstracker-api/utils"
	"regexp"
)

var reStripTrailingSlash = regexp.MustCompile(`/$`)
var fileStoreAbsPath string
var loggingFilePath string

func fileLogging() (f *os.File) {
	log.SetFlags(log.Ldate | log.Ltime)
	f, err := os.OpenFile(
		loggingFilePath,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666)
	if err != nil {
		log.Fatalf("Error opening file %v", err)
	}
	log.SetOutput(f)
	return
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: absolute file storage path missing")
		os.Exit(1)
	}

	dirExists, _ := utils.PathExists(os.Args[1])
	if !dirExists {
		log.Fatalf("Cannot open directory %s", os.Args[1])
	}
	workingDirectory := reStripTrailingSlash.ReplaceAllString(
		path.Clean(os.Args[1]), "") + "/"
	os.Chdir(workingDirectory)
	loggingFilePath = workingDirectory + "receipts-api.log"
	storeReceiptsDirAbsPath := workingDirectory + external.UPLOAD_DIRECTORY

	f := fileLogging()
	defer f.Close()

	log.Printf("Using %s directory to store receipts\n",
		storeReceiptsDirAbsPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpserver.ApiHandler)
	log.Printf("Listening on port %q\n", external.PORT)
	if err := http.ListenAndServe(external.PORT, mux); err != nil {
		log.Fatalf("Cannot listen on port %q: %q", external.PORT, err)
	}
}
