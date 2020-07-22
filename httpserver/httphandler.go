package httpserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"receiptstracker-api/dbengine"
	"receiptstracker-api/external"
	"receiptstracker-api/utils"
)

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("Incoming %s [%v] connection from %s with size %d bytes",
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
		r.Body = http.MaxBytesReader(w, r.Body, external.MAX_FILE_SIZE+512)
		if err := r.ParseMultipartForm(external.MAX_FILE_SIZE); err != nil {
			log.Printf("ERROR: parsing form failed: %v", err)
			userErrMsg := "Couldn't parse form or mandatory value(s) missing"
			fmt.Fprint(w, userErrMsg+"\r\n")
			return
		}

		tags := NormaliseTags(r.FormValue("tags"))
		log.Printf("Parsed tags: %v", *tags)

		formFile, formFileHeaders, err := r.FormFile("file")
		if err != nil {
			log.Printf("ERROR: no file included")
			fmt.Fprint(w, "Missing 'file' parameter\r\n")
			return
		}
		if utils.IsAllowedFileExt(formFileHeaders.Filename) == false {
			log.Printf("ERROR: file extension not allowed: %s",
				formFileHeaders.Filename)
			fmt.Fprintf(w, "ERROR: File extension not allowed. Allowed extensions: %v\r\n",
				external.AllowedExtensions)
			return
		}
		// Get binary from form
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
		log.Printf("Hash of incoming filename %s is %s",
			formFileHeaders.Filename,
			filename)
		// Write or try to write file
		writePath := filepath.Join(external.UPLOAD_DIRECTORY, filename)
		duplicate, err := utils.PathExists(writePath)
		if duplicate {
			fmt.Fprint(w, "Error: receipt already archived\r\n")
			log.Printf("ERROR: Receipt already archived: %v", err)
			return
		}
		err = ioutil.WriteFile(writePath, binFile, 0600)
		if err != nil {
			log.Printf("ERROR: writing file %s: %v", writePath, err)
			fmt.Fprint(w, "Failed to save file\r\n")
			return
		}
		log.Printf("Wrote file to %s", writePath)

		var expiryDate string = ""
		var purchaseDate string = ""
		purchaseDateTmp, err := ParsePurchaseDate(tags)
		if err != nil {
			log.Printf("WARNING: no purchase date: %v", err)
		} else {
			purchaseDate = purchaseDateTmp.Format("2006-01-02")
			log.Printf("Found and parsed purchase date: %s",
				purchaseDate)

			expiryDateTmp, err := ParseExpiryDate(tags, purchaseDateTmp)
			if err != nil {
				log.Printf("WARNING: no expiry date: %v", err)
			} else {
				expiryDate = expiryDateTmp.Format("2006-01-02")
				log.Printf("Found and parsed expiry date: %s",
					expiryDate)
			}
		}

		// XXX Handle format here
		receiptId, err := dbengine.InsertReceipt(
			ctx,
			filename,
			purchaseDate,
			expiryDate)
		if err != nil {
			// TODO Show error to user
			return
		}
		tagsWriteSucceed := dbengine.InsertTags(ctx, *tags)
		if tagsWriteSucceed == false {
			fmt.Fprint(w, "Failed to write tags\r\n")
			return
		}
		tagAssociationCount, err := dbengine.InsertReceiptTagAssociation(
			ctx,
			receiptId,
			*tags)
		if err != nil {
			fmt.Fprint(w, "Failed to write receipt ID <-> tag IDs associations")
			return
		}
		log.Printf("Wrote %d number of associations for receipt ID %d",
			tagAssociationCount,
			receiptId)

		doneMsg := fmt.Sprintf("Storing of receipt %s completed",
			filename)
		log.Print(doneMsg)
		fmt.Fprint(w, doneMsg+"\r\n")
	default:
		fmt.Fprint(w, "Supported methods: GET, POST\r\n")
		return
	}
}
