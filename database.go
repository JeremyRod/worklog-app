package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type Entry struct {
	hours     time.Duration
	projCode  string
	desc      string
	startTime time.Time
	endTime   time.Time
	date      time.Time
}

type EntryRow struct {
	entry   Entry
	entryId int
}

func (d *Database) SaveEntry(entry *EntryRow) error {
	tx, err := d.db.Begin()
	if err != nil {
		fmt.Println(err)
	}
	stmt, err := tx.Prepare("insert into worklog(hours, desc, projcode, starttime, endtime, date) values(?, ?, ?, ?, ?, ?)")

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(entry.entry.hours,
		entry.entry.desc, entry.entry.projCode, entry.entry.startTime,
		entry.entry.endTime, entry.entry.date)
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

	sqlStmt := `
	create table worklog (id integer not null primary key, hours time, desc text, starttime time not null, endtime time, projcode text not null, date date not null
	CHECK (hours IS NOT NULL OR endtime IS NOT NULL));
	delete from worklog;
	`
	_, err := d.db.Exec(sqlStmt)
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
	if err != nil {
		return errors.New("database broke")
	}
	d.db = db
	_, err = d.db.Query("select * from worklog;")
	if err != nil {
		d.CreateDatabase()
	}
	return nil
}

func (d *Database) CloseDatabase() {
	d.db.Close()
}

func (e *EntryRow) FillData(m model) {
	timeFmt := "15:04"
	dateFmt := "02/01/2006"
	var err error
	e.entry.hours, err = time.ParseDuration(m.inputs[hours].Value())
	if err != nil {
		fmt.Println(err)
	}
	e.entry.startTime, err = time.Parse(timeFmt, m.inputs[startTime].Value())
	if err != nil {
		fmt.Println(err)
	}
	e.entry.endTime, err = time.Parse(timeFmt, m.inputs[endTime].Value())
	if err != nil {
		fmt.Println(err)
	}
	e.entry.date, err = time.Parse(dateFmt, m.inputs[date].Value())
	if err != nil {
		fmt.Println(err)
	}
	e.entry.projCode = m.inputs[code].Value()
	e.entry.desc = m.inputs[desc].Value()
	if e.entry.hours == 0 {
		e.entry.hours = time.Duration(e.entry.endTime.Sub(e.entry.startTime).Minutes())
	}
}
