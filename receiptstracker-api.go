package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type LogEntry string

type HtmlRenderElement struct {
	LogEntries   []LogEntry
	NextGreasing string
}

const (
	port            string = ":8081"
	maxFileSite     int    = 16 * 1024 * 1024
	uploadDirectory string = "uploads"
)

var allowedExtensions []string = []string{
	"gif",
	"jpg",
	"jpeg",
	"png",
	"tiff",
}

var purchaseDatePat = regexp.MustCompile(`^[0-9]{4}\-[0-9]{1,2}\-[0-9]{1,2}$`)
var expiryDatePat = regexp.MustCompile(`^[0-9]+_(day|month|year)s?$`)
var reStripTrailingSlash = regexp.MustCompile(`/$`)
var fileStoreAbsPath string
var loggingFileAbsPath string

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

func api(w http.ResponseWriter, r *http.Request) {
	log.Printf("Incoming %s [%s] connection from %s with size %d bytes",
		r.Method,
		r.Header,
		r.RemoteAddr,
		r.ContentLength)

	switch r.Method {
	case "GET":
		t := template.New("receipt_template")
		t, err := template.ParseFiles("form.html")
		if err != nil {
			log.Fatalf("Couldn't load form.html: %v", err)
		}

		filledInData := HtmlRenderElement{nil, ""}
		t.Execute(w, filledInData)
	case "POST":
		if err := r.ParseForm(); err != nil {
			log.Printf("Error parsing form\n")
			return
		}

		// filename = secure_filename(received_file.filename)
		// file_binary = received_file.stream.read()
		// filename_hash = hashlib.sha256(file_binary).hexdigest()
		// ext = os.path.splitext(filename)[-1].strip(".").lower()
		// outfile = //{filename_hash}.{ext}")
		// purchase_date = parse_purchase_date(tags)
		// expiry_date = parse_expiry_date(purchase_date, tags)

		datePicker := template.HTMLEscapeString(r.FormValue("datePicker"))
		if len(datePicker) < 1 {
			fmt.Fprintf(w, "Error, date not appropriate (yyyy-mm-dd)\r\n")
			return
		}

		tmp := fmt.Sprintf("Receipt [%s] '%s' saved\r\n", "daA", "boo")
		fmt.Fprintf(w, tmp)
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
	log.Printf("Listening on port %q\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Cannot listen on port %q: %q", port, err)
	}
}
