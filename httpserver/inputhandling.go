package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"receiptstracker-api/utils"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var expiryDatePat = regexp.MustCompile(`^[0-9]+_(day|month|year)s?$`)
var purchaseDatePat = regexp.MustCompile(`^[0-9]{4}\-[0-9]{1,2}\-[0-9]{1,2}$`)

func ParsePurchaseDate(tags *[]string) (time.Time, error) {
	for i, t := range *tags {
		found := purchaseDatePat.FindString(t)
		if found == "" {
			continue
		}

		dtime, err := time.Parse("2006-01-02", t)
		if err != nil {
			log.Printf("ERROR: while parsing date '%s'", t)
			continue
		}
		*tags = utils.DeleteFromSlice(*tags, i)
		return dtime, nil
	}
	return time.Time{}, errors.New("Couldn't find or parse date")
}

// ParseExpiryDate goes through the tags and returns
// the match of first occurrence of expiry date.
func ParseExpiryDate(tags *[]string, startDate time.Time) (time.Time, error) {
	for i, t := range *tags {
		found := expiryDatePat.FindString(t)
		if found == "" {
			continue
		}

		parsedNumber := regexp.MustCompile(`[0-9]+`).FindString(t)
		if parsedNumber == "" {
			errorMsg := "Found day|month|year %s but couldn't parse numbers"
			log.Printf(errorMsg, t)
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
			*tags = utils.DeleteFromSlice(*tags, i)
			return startDate.AddDate(0, 0, numberVal), nil
		}
		if months != "" {
			*tags = utils.DeleteFromSlice(*tags, i)
			return startDate.AddDate(0, numberVal, 0), nil
		}
		if years != "" {
			*tags = utils.DeleteFromSlice(*tags, i)
			return startDate.AddDate(numberVal, 0, 0), nil
		}
	}
	return time.Time{}, errors.New("No expiry time found")
}

func CalculateFileHash(binFile []byte,
	formFileHeaders *multipart.FileHeader) (string, error) {
	if len(binFile) == 0 {
		return "", errors.New("Empty file")
	}

	tmp := filepath.Ext(formFileHeaders.Filename)
	fileExt := strings.Trim(tmp, ".")
	fileExt = strings.ToLower(fileExt)

	tmpHash := sha256.Sum256(binFile)
	fileHash := hex.EncodeToString(tmpHash[:])

	fullFileName := fmt.Sprintf("%s.%s", fileHash, fileExt)

	return fullFileName, nil
}

func NormaliseTags(tags string) *[]string {
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
