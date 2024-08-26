package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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

func (d *Database) DeleteEntry(entry EntryRow) error {
	sqlstmt := fmt.Sprintf(`delete from worklog where id = %d`, entry.entryId)
	_, err := d.db.Exec(sqlstmt)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (d *Database) ModifyEntry(e EntryRow) error {
	// TODO: Could optimise to only update what is changed
	sqlstmt := fmt.Sprintf(`Update worklog set desc = ?, 
							hours = ?, 
							projcode = ?, 
							date = ?, 
							starttime = ?, 
							endtime = ? where id = ?;`)
	tx, err := d.db.Begin()
	if err != nil {
		fmt.Println(err)
	}
	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		fmt.Println(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(e.entry.desc, e.entry.hours,
		e.entry.projCode, e.entry.date, e.entry.startTime,
		e.entry.endTime, e.entryId)
	if err != nil {
		fmt.Println(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (d *Database) QueryEntries(m *model) ([]EntryRow, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if m.id == 0 {
		rows, err = d.db.Query("select date, id, projcode, hours, desc from worklog order by id desc limit 10")
	} else {
		rows, err = d.db.Query(fmt.Sprintf("select date, id, projcode, hours, desc from worklog order by id desc limit 10, %d", m.id))
	}
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	ents := []EntryRow{}
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.entry.date, &ent.entryId, &ent.entry.projCode, &ent.entry.hours, &ent.entry.desc)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(ent.entryId, ent.entry.projCode)
		ents = append(ents, ent)
		m.id = ent.entryId
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return ents, nil
}

func (d *Database) QueryEntry(e EntryRow) (EntryRow, error) {
	var (
		rows *sql.Rows
		err  error
	)

	rows, err = d.db.Query("select date, id, projcode, hours, desc, starttime, endtime from worklog where id = %d", e.entryId)
	defer rows.Close()
	var ent EntryRow
	for rows.Next() {
		err = rows.Scan(&ent.entry.date, &ent.entryId, &ent.entry.projCode, &ent.entry.hours, &ent.entry.desc, &ent.entry.startTime, &ent.entry.endTime)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(ent.entryId, ent.entry.projCode)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return ent, nil
}

func (d *Database) CreateDatabase() error {

	sqlStmt := `
	create table worklog (id integer not null primary key, hours time not null, desc text, starttime time not null, endtime time, projcode text not null, date date not null
	delete from worklog;
	`
	_, err := d.db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return fmt.Errorf("db stmt fail %q: %s", err, sqlStmt)
	}
	return nil
}

// TODO: Seed database for better tests.
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

func (e *EntryRow) FillData(inputs []textinput.Model) error {
	timeFmt := "15:04"
	dateFmt := "02/01/2006"
	var err error
	e.entry.hours, err = time.ParseDuration(inputs[hours].Value())
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	e.entry.startTime, err = time.Parse(timeFmt, inputs[startTime].Value())
	if err != nil && !e.entry.startTime.IsZero() {
		return fmt.Errorf("%s", err)
	}
	e.entry.endTime, err = time.Parse(timeFmt, inputs[endTime].Value())
	if err != nil && !e.entry.endTime.IsZero() {
		return fmt.Errorf("%s", err)
	}
	e.entry.date, err = time.Parse(dateFmt, inputs[date].Value())
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	e.entry.projCode = inputs[code].Value()
	e.entry.desc = inputs[desc].Value()
	if e.entry.hours == 0 {
		e.entry.hours = time.Duration(e.entry.endTime.Sub(e.entry.startTime))
	}
	return nil
}
