package dbengine

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"receiptstracker-api/utils"

	_ "github.com/mattn/go-sqlite3"
)

const sqlSchema = `CREATE TABLE receipt (
        id INTEGER PRIMARY KEY,
        filename VARCHAR NOT NULL,
        purchase_date DATE,
        expiry_date DATE,
        ocr_text VARCHAR,
        UNIQUE (filename)
);
CREATE TABLE tag (
        id INTEGER PRIMARY KEY,
        tag VARCHAR,
        UNIQUE (tag)
);
CREATE TABLE receipt_tag_association (
        id INTEGER PRIMARY KEY,
        receipt_id INTEGER,
        tag_id INTEGER,
        FOREIGN KEY(receipt_id) REFERENCES receipt (id),
        FOREIGN KEY(tag_id) REFERENCES tag (id)
);
`

var (
	DbConn *sql.DB
)

func InitDbConn(db *sql.DB) {
	DbConn = db
}

func CreateSchema(db *sql.DB) {
	_, err := db.Exec(sqlSchema)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: schema creation failed: %v", err)
		log.Fatal(errMsg)
	}
}

func ConnectAndInit(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	// Create schema if doesn't exist
	if exists, _ := utils.PathExists(dbPath); exists == false {
		CreateSchema(db)
	}

	return db
}

// InsertReceipt returns true if insert succeeds, false otherwise
func InsertReceipt(
	ctx context.Context,
	db *sql.DB,
	filename string,
	purchaseDate string,
	expiryDate string) (int64, error) {
	stmt, err := db.PrepareContext(ctx, `
INSERT OR IGNORE INTO receipt(
	filename,
	purchase_date,
	expiry_date
) VALUES (
	:filename,
	:purchase_date,
	:expiry_date);`)

	defer stmt.Close()

	res, err := stmt.ExecContext(ctx,
		sql.Named("filename", filename),
		sql.Named("purchase_date", purchaseDate),
		sql.Named("expiry_date", expiryDate),
	)
	if err != nil {
		log.Printf("ERROR: receipt insert failed: %v", err)
		return 0, err
	}

	receiptId, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERROR: failed to get last inserted id: %v", err)
		return 0, err
	}
	return receiptId, nil
}

func InsertTags(ctx context.Context, db *sql.DB, tags []string) bool {
	rawSql := "INSERT OR IGNORE INTO tag (tag) VALUES "
	values := []interface{}{}

	for _, tag := range tags {
		rawSql += "(?),"
		values = append(values, tag)
	}
	// Remove comma postfix
	rawSql = rawSql[0 : len(rawSql)-1]
	stmt, err := db.PrepareContext(ctx, rawSql)
	if err != nil {
		log.Printf("ERROR: preparing statement for tags failed: %v",
			err)
		return false
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, values...)
	if err != nil {
		log.Printf("ERROR: inserting tags failed: %v", err)
	} else {
		return true
	}

	return false
}

func InsertReceiptTagAssociation(
	ctx context.Context,
	db *sql.DB,
	receiptId int64,
	tags []string) (int64, error) {
	values := []interface{}{}
	rawSql := "INSERT OR IGNORE INTO receipt_tag_association (receipt_id, tag_id) VALUES "

	tagIds := getTagsIds(ctx, db, tags)
	for tagId, _ := range tagIds {
		values = append(values, receiptId)
		values = append(values, tagId)
		rawSql += "(?, ?),"
	}
	// Remove comma postfix
	rawSql = rawSql[0 : len(rawSql)-1]

	stmt, err := db.PrepareContext(ctx, rawSql)
	if err != nil {
		log.Printf("ERROR: preparing statement for tag ids failed: %v", err)
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, values...)
	if err != nil {
		log.Printf("ERROR: failed to insert receipt tag association: %v", err)
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		log.Printf("ERROR: failed to get affected row in receipt tag association: %v", err)
		return 0, err
	}

	return affected, nil
}

func getTagsIds(ctx context.Context, db *sql.DB, tags []string) map[int64]string {
	rawSql := "SELECT id, tag FROM tag WHERE tag IN ("
	values := []interface{}{}

	for _, tag := range tags {
		rawSql += "?,"
		values = append(values, tag)
	}
	// Remove comma postfix
	rawSql = rawSql[0 : len(rawSql)-1]
	rawSql += ");"
	stmt, err := db.PrepareContext(ctx, rawSql)
	if err != nil {
		log.Printf("ERROR: preparing statement for tag ids failed: %v", err)
		return map[int64]string{}
	}
	defer stmt.Close()

	tagIds := make(map[int64]string, 0)
	rows, err := stmt.QueryContext(ctx, values...)
	if err != nil {
		log.Printf("ERROR: getting tag ids failed: %v", err)
		return map[int64]string{}
	}
	for rows.Next() {
		var tagId int64
		var tag string
		err := rows.Scan(&tagId, &tag)
		if err != nil {
			log.Printf("ERROR: failed to get tagId: %v", err)
			continue
		}

		tagIds[tagId] = tag
	}

	return tagIds
}
