package httpserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"receiptstracker-api/external"
	"receiptstracker-api/utils"
	"reflect"
	"time"
)

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Incoming %s [%s] connection from %s with size %d bytes",
		r.Method,
		r.Header,
		r.RemoteAddr,
		r.ContentLength)

	switch r.Method {
	case "GET":
		if err := LoadPage(w, r); err != nil {
			log.Printf("ERROR: %s", err)
			return
		}
	case "POST":
		// Limit request's maximum size to 16.5 MB
		r.Body = http.MaxBytesReader(w, r.Body, (16*1024*1024)+512)
		if err := r.ParseMultipartForm(16 * 1024 * 1024); err != nil {
			log.Printf("ERROR: parsing form failed: %v", err)
			userErrMsg := "Couldn't parse form or mandatory value(s) missing"
			fmt.Fprintf(w, userErrMsg+"\r\n")
			return
		}

		tags := NormaliseTags(r.FormValue("tags"))

		formFile, formFileHeaders, err := r.FormFile("file")
		if err != nil {
			log.Printf("ERROR: no file included")
			fmt.Fprintf(w, "Missing 'file' parameter\r\n")
			return
		}

		if utils.IsAllowedFileExt(formFileHeaders.Filename) == false {
			log.Printf("ERROR: file extension not allowed: %s",
				formFileHeaders.Filename)
			fmt.Fprintf(w, "ERROR: File extension not allowed. Allowed extensions: %v\r\n",
				external.AllowedExtensions)
			return
		}

		// Get the binary of the form file
		binFile, err := ioutil.ReadAll(formFile)
		if err != nil {
			log.Printf("ERROR: reading file %s failed: %v",
				formFileHeaders.Filename,
				err)
			fmt.Fprint(w, "Error while reading file binary\r\n")
			return
		}

		filename, err := CalculateFileHash(binFile, formFileHeaders)
		if err != nil {
			log.Printf("ERROR: %s", err)
			fmt.Fprintf(w, "%s\r\n", err)
			return
		}
		log.Printf("Hash for incoming filename %s is %s",
			formFileHeaders.Filename,
			filename)

		// Write or try to write file
		writePath := filepath.Join(external.UPLOAD_DIRECTORY, filename)
		err = ioutil.WriteFile(writePath, binFile, 0600)
		if err != nil {
			log.Printf("Error writing file %s: %v", writePath, err)
			fmt.Fprintf(w, "Failed to save file \r\n")
			return
		}
		log.Printf("Wrote file to %s", writePath)

		purchaseDate, err := ParsePurchaseDate(tags)
		if err != nil {
			log.Printf("WARNING: no purchase date: %v", err)
		}
		expiryDate := ParseExpiryDate(tags, purchaseDate)
		if reflect.DeepEqual(expiryDate, time.Time{}) {
			log.Printf("WARNING: no expiry date %s: %v",
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
