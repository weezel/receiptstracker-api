package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	PORT             string = ":8081"
	MAX_FILE_SIZE    int    = 16 * 1024 * 1024
	UPLOAD_DIRECTORY string = "uploads"
)

var purchaseDatePat = regexp.MustCompile(`^[0-9]{4}\-[0-9]{1,2}\-[0-9]{1,2}$`)
var expiryDatePat = regexp.MustCompile(`^[0-9]+_(day|month|year)s?$`)
var reStripTrailingSlash = regexp.MustCompile(`/$`)
var fileStoreAbsPath string
var loggingFileAbsPath string

var allowedExtensions []string = []string{
	"gif",
	"jpg",
	"jpeg",
	"png",
	"tiff",
}

func deleteFromSlice(a []string, i int) []string {
	return append(a[:i], a[i+1:]...)
}

func isAllowerFileExt(fname string) bool {
	fileExt := strings.ToLower(filepath.Ext(fname))
	for _, ext := range allowedExtensions {
		if ext != fileExt {
			return false
		}
	}
	return true
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func parsePurchaseDate(tags *[]string) (time.Time, error) {
	for i, t := range *tags {
		found := purchaseDatePat.FindString(t)
		if found != "" {
			dtime, err := time.Parse("2006-01-02", t)
			if err != nil {
				log.Printf("Error while parsing date '%s'", t)
				return time.Time{}, err
			}
			*tags = deleteFromSlice(*tags, i)
			return dtime, nil
		}
	}
	return time.Time{}, errors.New("Couldn't find or parse date")

}

func parseExpiryDate(tags *[]string, startDate time.Time) (time.Time, error) {
	for i, t := range *tags {
		found := expiryDatePat.FindString(t)
		if found == "" {
			continue
		}

		parsedNumber := regexp.MustCompile(`[0-9]+`).FindString(t)
		if parsedNumber == "" {
			log.Printf("Found day|month|year %s but couldn't parse numbers", t)
			continue
		}
		var numberVal int
		numberVal, err := strconv.Atoi(parsedNumber)
		if err != nil {
			log.Printf("Error while parsing %s as a number", parsedNumber)
			continue
		}

		days := regexp.MustCompile(`days?$`).FindString(t)
		months := regexp.MustCompile(`months?$`).FindString(t)
		years := regexp.MustCompile(`years?$`).FindString(t)
		if days != "" {
			*tags = deleteFromSlice(*tags, i)
			return startDate.AddDate(0, 0, numberVal), nil
		} else if months != "" {
			*tags = deleteFromSlice(*tags, i)
			return startDate.AddDate(0, numberVal, 0), nil
		} else if years != "" {
			*tags = deleteFromSlice(*tags, i)
			return startDate.AddDate(numberVal, 0, 0), nil
		}
	}
	return time.Time{}, nil
}

// TODO Parameterize out writer
func saveFile(formFile *multipart.File, formFileHeaders *multipart.FileHeader) (string, error) {

	binFile, err := ioutil.ReadAll(*formFile)
	if err != nil {
		log.Printf("Error while reading file %v: %v",
			formFileHeaders,
			err)
		return "", err
	}
	tmpHash := sha256.Sum256(binFile)
	fileHash := hex.EncodeToString(tmpHash[:])
	fileExt := filepath.Ext(formFileHeaders.Filename)

	writePath := filepath.Join(UPLOAD_DIRECTORY, fileHash)
	if err = ioutil.WriteFile(
		writePath,
		binFile, 0600); err != nil {
		log.Printf("Error writing file %s", fileHash)
	}
	log.Printf("Wrote file to %s", writePath)

	fullFileName := fmt.Sprintf("%s.%s", fileHash, fileExt)
	return fullFileName, nil
}

func loadPage(w http.ResponseWriter, r *http.Request) {
	page, err := ioutil.ReadFile("resources/send.html")
	if err != nil {
		log.Printf("Error loading page send.hmtl")
		fmt.Fprintf(w, "Could't load form\r\n")
		return
	}
	fmt.Fprintf(w, "%s", page)

}

func normaliseTags(tags string) *[]string {
	keys := make(map[string]bool)
	list := &[]string{}
	p := regexp.MustCompile(`\s+`)
	tagsSingleSpaces := p.ReplaceAllString(tags, " ")

	// Remove duplicates
	for _, entry := range strings.Split(tagsSingleSpaces, " ") {
		if _, value := keys[entry]; !value {
			trimmed := strings.Trim(entry, " ")
			if trimmed == "" {
				continue
			}
			keys[entry] = true
			*list = append(*list, trimmed)
		}
	}
	return list
}

func api(w http.ResponseWriter, r *http.Request) {
	log.Printf("Incoming %s [%s] connection from %s with size %d bytes",
		r.Method,
		r.Header,
		r.RemoteAddr,
		r.ContentLength)

	switch r.Method {
	case "GET":
		loadPage(w, r)
	case "POST":
		// Limit request's maximum size to 16.5 MB
		r.Body = http.MaxBytesReader(w, r.Body, (16*1024*1024)+512)
		if err := r.ParseMultipartForm(16 * 1024 * 1024); err != nil {
			log.Printf("Error: parsing form failed: %v", err)
			userErrMsg := "Couldn't parse form or mandatory value(s) missing"
			fmt.Fprintf(w, userErrMsg+"\r\n")
			return
		}

		tags := normaliseTags(r.FormValue("tags"))

		formFile, formFileHeaders, err := r.FormFile("file")
		if err != nil {
			log.Printf("Error: no file included")
			fmt.Fprintf(w, "Missing 'file' parameter\r\n")
			return
		}
		filename, err := saveFile(&formFile, formFileHeaders)
		if err != nil {
			log.Printf("Error: failed to save file %s", filename)
			fmt.Fprintf(w, "Failed to save file \r\n")
			return
		}
		log.Printf("Hash for incoming filename %s is %s",
			formFileHeaders.Filename,
			filename)

		purchaseDate, err := parsePurchaseDate(tags)
		if err != nil {
			fmt.Printf("Error while parsing purchase date: %v", err)
		}
		expiryDate, err := parseExpiryDate(tags, purchaseDate)
		if err != nil {
			fmt.Printf("Error while parsing expiry date %s: %v",
				expiryDate,
				err)
		}

		doneMsg := fmt.Sprintf("Storing of receipt %s completed",
			filename)
		log.Print(doneMsg)
		fmt.Fprintf(w, doneMsg+"\r\n")
	default:
		fmt.Fprintf(w, "Supported methods: GET, POST\r\n")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: absolute file storage path missing")
		os.Exit(1)
	}

	dirExists, _ := pathExists(os.Args[1])
	if !dirExists {
		log.Fatalf("Cannot open directory %s", os.Args[1])
	}
	workingDirectory := reStripTrailingSlash.ReplaceAllString(
		path.Clean(os.Args[1]), "") + "/"
	os.Chdir(workingDirectory)
	loggingFileAbsPath = workingDirectory + "receipts-api.log"
	storeReceiptsDirAbsPath := workingDirectory + "img/"

	log.SetFlags(log.Ldate | log.Ltime)
	f, err := os.OpenFile(
		loggingFileAbsPath,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666)
	if err != nil {
		log.Fatalf("Error opening file %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	fmt.Printf("Using %s directory to store receipts\n",
		storeReceiptsDirAbsPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/", api)
	log.Printf("Listening on port %q\n", PORT)
	if err := http.ListenAndServe(PORT, mux); err != nil {
		log.Fatalf("Cannot listen on port %q: %q", PORT, err)
	}
}
