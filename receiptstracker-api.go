package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"receiptstracker-api/dbengine"
	"receiptstracker-api/external"
	"receiptstracker-api/httpserver"
	"receiptstracker-api/utils"
	"regexp"
	"syscall"
)

var (
	fileStoreAbsPath     string
	loggingFilePath      string
	reStripTrailingSlash *regexp.Regexp
	logFile              *os.File
)

func init() {
	reStripTrailingSlash = regexp.MustCompile(`/$`)
}

func fileLogging() (f *os.File) {
	log.SetFlags(log.Ldate | log.Ltime)
	f, err := os.OpenFile(
		loggingFilePath,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0600)
	if err != nil {
		log.Fatalf("Error opening file %v", err)
	}
	log.SetOutput(f)
	return
}

func signalHandler(signalCh chan os.Signal, doneCh chan struct{}) {
	for {
		select {
		case s := <-signalCh:
			fmt.Println("Shutting down...")
			log.Printf("Received signal: %d (%s)", s, s)

			if err := logFile.Sync(); err != nil { // FIXME
				log.Printf("ERROR: syncing logfile failed: %v", err)
			}

			dbengine.ShutdownDb()

			if err := logFile.Close(); err != nil { // FIXME
				log.Printf("ERROR: Failed to close logfile: %v", err)
			}
			os.Exit(0)
			doneCh <- struct{}{}
		}
	}
}

func connectAndInitDb(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	if exists, _ := utils.PathExists(dbPath); exists == false {
		dbengine.CreateSchema(db)
	}

	return db
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: absolute file storage path missing")
		os.Exit(1)
	}
	doneCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go signalHandler(signalCh, doneCh)

	dirExists, _ := utils.PathExists(os.Args[1])
	if !dirExists {
		log.Fatalf("Cannot open directory %s", os.Args[1])
	}
	workingDirectory := reStripTrailingSlash.ReplaceAllString(
		path.Clean(os.Args[1]), "") + "/"
	if err := os.Chdir(workingDirectory); err != nil {
		log.Printf("ERROR: chdir() failed: %v", err)
	}
	loggingFilePath = workingDirectory + "receipts-api.log"
	storeReceiptsDirAbsPath := workingDirectory + external.UPLOAD_DIRECTORY

	logFile := fileLogging()
	defer logFile.Close()

	db := connectAndInitDb("receipts.db")
	dbengine.UpdateDbRef(db)
	db = nil // Remove reference from the main
	log.Printf("Database ready")

	log.Printf("Using %s directory to store receipts\n",
		storeReceiptsDirAbsPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpserver.ApiHandler)
	log.Printf("Listening on port %q\n", external.PORT)
	if err := http.ListenAndServe(external.PORT, mux); err != nil {
		log.Fatalf("Cannot listen on port %q: %q", external.PORT, err)
	}

	<-doneCh
}
