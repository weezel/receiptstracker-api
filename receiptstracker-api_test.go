package main

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParsePurchaseDate(t *testing.T) {
	tags := []string{"2019-02-06", "testing", "not_a_date"}
	expectedDate := time.Date(2019, 2, 6, 0, 0, 0, 0, time.UTC)
	parsedDate, err := parsePurchaseDate(&tags)
	log.Printf("expectedDate: %v", expectedDate)
	log.Printf("parsedDate: %v", parsedDate)
	assert.NotNil(t, "parsedDate had an error set", err)
	assert.Equal(t, expectedDate, parsedDate)

	expectedTags := []string{"testing", "not_a_date"}
	assert.Equal(t, expectedTags, tags)
}
