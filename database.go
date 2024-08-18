package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
	id int
}

type Entry struct {
	hours     float64
	projCode  string
	desc      string
	startTime time.Time
	endTime   time.Time
}

type EntryRow struct {
	entry   Entry
	entryId int
}

func (d *Database) SaveEntry(entry *EntryRow) error {
	tx, err := d.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into worklog(id, hours, desc, projcode) values(?, ?, ?, ?)")

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(entry.entryId, entry.entry.hours, entry.entry.desc, entry.entry.projCode)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Database) DeleteEntry(entry *EntryRow) error {
	sqlstmt := fmt.Sprintf(`delete from worklog where id = %d`, entry.entryId)
	_, err := d.db.Exec(sqlstmt)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (d *Database) ModifyEntry(entry *EntryRow) error {
	return nil
}

func (d *Database) QueryEntries(id int) ([]EntryRow, error) {
	rows, err := d.db.Query("select id, projcode, hours from worklog")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	ents := []EntryRow{}
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.entryId, &ent.entry.projCode, &ent.entry.hours)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(ent.entryId, ent.entry.projCode)
		ents = append(ents, ent)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return ents, nil
}

func (d *Database) CreateDatabase() error {
	os.Remove("./worklog.db")

	db, err := sql.Open("sqlite3", "./worklog.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
	create table worklog (id integer not null primary key, hours float,
	starttime time not null, endtime time, desc text,
	projcode text not null, check(hours is not null or endtime is not null));
	delete from worklog;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return fmt.Errorf("db stmt fail %q: %s", err, sqlStmt)
	}
	return nil
}

func (d *Database) SeedDatabase() error {
	tx, err := d.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into worklog(id, name) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("こんにちは世界%03d", i))
		if err != nil {
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Database) OpenDatabase() error {
	db, err := sql.Open("sqlite3", "./worklog.db")
	d.db = db
	if err != nil {
		return errors.New("database not exist")
	}
	return nil
}

func (d *Database) CloseDatabase() {
	d.db.Close()
}
