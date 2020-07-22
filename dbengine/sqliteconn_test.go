package dbengine

import (
	"context"
	"database/sql"
	"log"
	"reflect"
	"sort"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// This is only needed to get order right and to therefore to compare things.
type pair struct {
	receiptId int64
	tagId     int64
}
type pairList []pair

func (p pairList) Len() int           { return len(p) }
func (p pairList) Less(i, j int) bool { return p[i].tagId < p[j].tagId }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func TestInsertTags(t *testing.T) {
	expectedTags := []string{"yo", "dawg"}
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(5)*time.Second)
	defer cancel()

	UpdateDbRef(memDb)
	CreateSchema(memDb)

	type args struct {
		ctx  context.Context
		tags []string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Insert tags",
			args{ctx, expectedTags},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InsertTags(tt.args.ctx, tt.args.tags)
			if got != tt.want {
				t.Errorf("%s: InsertTags() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}

	// Ensure that inserted elements are found from the db
	insertedTags := make([]string, 0)
	rows, _ := memDb.Query("SELECT tag FROM tag;")
	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			t.Errorf("Failed to get row data: %v", err)
		}
		insertedTags = append(insertedTags, tag)
	}
	if !reflect.DeepEqual(insertedTags, expectedTags) {
		t.Errorf("ERROR: mismatch in insertedTags: %v", insertedTags)
	}

	ShutdownDb()
}

func TestgetTagsIds(t *testing.T) {
	expectedTags := []string{"computershop", "laptop", "2019-05-15"}
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(5)*time.Second)
	defer cancel()

	UpdateDbRef(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`INSERT INTO tag (tag) VALUES ('computershop'), ('laptop'), ('2019-05-15');`)
	if err != nil {
		log.Fatalf("Unexpected error on SQL INSERT: %v", err)
	}

	type args struct {
		ctx  context.Context
		tags []string
	}
	tests := []struct {
		name string
		args args
		want map[int]string
	}{
		{
			"Laptop purchase",
			args{ctx, expectedTags},
			map[int]string{
				1: expectedTags[0],
				2: expectedTags[1],
				3: expectedTags[2],
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTagsIds(tt.args.ctx, tt.args.tags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: GetTagsIds() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}

	ShutdownDb()
}

func TestInsertReceiptTagAssociation(t *testing.T) {
	expectedTags := []string{"computershop", "laptop", "2019-05-15"}
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(5)*time.Second)
	defer cancel()

	UpdateDbRef(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`INSERT INTO tag (tag) VALUES ('computershop'), ('laptop'), ('2019-05-15');`)
	if err != nil {
		log.Fatalf("Unexpected error on SQL INSERT: %v", err)
	}

	type args struct {
		ctx       context.Context
		receiptId int64
		tags      []string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			"Simple association",
			args{ctx, 0, expectedTags},
			3,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InsertReceiptTagAssociation(
				tt.args.ctx,
				tt.args.receiptId,
				tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: InsertReceiptTagAssociation() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%s: InsertReceiptTagAssociation() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}

	// Ensure that inserted tags are found from the db
	insertedTags := make([]string, 0)
	tagRows, _ := memDb.Query("SELECT tag FROM tag;")
	for tagRows.Next() {
		var tag string
		err := tagRows.Scan(&tag)
		if err != nil {
			t.Errorf("Failed to get tagRow data: %v", err)
		}
		insertedTags = append(insertedTags, tag)
	}
	if !reflect.DeepEqual(insertedTags, expectedTags) {
		t.Errorf("ERROR: mismatch in insertedTags: %v", insertedTags)
	}

	// Ensure that inserted associations are found from the db
	insertedAssociations := make(pairList, 0)
	assocRows, _ := memDb.Query("SELECT receipt_id, tag_id FROM receipt_tag_association;")
	for assocRows.Next() {
		p := new(pair)
		err := assocRows.Scan(&p.receiptId, &p.tagId)
		if err != nil {
			t.Errorf("Failed to get assocRow data: %v", err)
		}
		insertedAssociations = append(insertedAssociations, *p)
	}
	sort.Sort(insertedAssociations)

	expectedAssociations := pairList{
		pair{receiptId: 0, tagId: 1},
		pair{receiptId: 0, tagId: 2},
		pair{receiptId: 0, tagId: 3},
	}

	if !reflect.DeepEqual(insertedAssociations, expectedAssociations) {
		t.Errorf("ERROR: mismatch in insertedAssociations: %v", insertedAssociations)
	}

	ShutdownDb()
}
