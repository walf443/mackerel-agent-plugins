package main

import (
	"database/sql"
	"testing"

	"github.com/erikstmartin/go-testdb"
)

func TestFetchStatDatabase(t *testing.T) {
	db, _ := sql.Open("testdb", "")

	columns := []string{"xact_commit", "xact_rollback", "blks_read", "blks_hit",
		"tup_returned", "tup_fetched", "tup_inserted", "tup_updated", "tup_deleted"}

	testdb.StubQuery(`
		select xact_commit, xact_rollback, blks_read, blks_hit,
			   tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted
		from pg_stat_database
	`, testdb.RowsFromCSVString(columns, `
	1,2,3,4,5,6,7,8,9
	10,20,30,40,50,60,70,80,90
	`))

	stat, err := FetchStatDatabase(db)

	if err != nil {
		t.Errorf("Expected no error, but got %s instead", err)
	}
	if err = db.Close(); err != nil {
		t.Errorf("Error '%s' was not expected while closing the database", err)
	}
	if stat["xact_commit"] != 11 {
		t.Error("should be 11")
	}
	if stat["blks_hit"] != 44 {
		t.Error("should be 44")
	}
	if stat["tup_returned"] != 55 {
		t.Error("should be 55")
	}

}
