package main

import (
	"bufio"
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
}

type Entry struct {
	hours     time.Duration
	projCode  string
	desc      string
	startTime time.Time
	endTime   time.Time
	date      time.Time
	notes     string
}

type EntryRow struct {
	entry   Entry
	entryId int
}

var ProjCodeToTask map[string]int // This is nil, reference before assignment will cause nil pointer issues

func (d *Database) SaveEntry(entry EntryRow) error {
	tx, err := d.db.Begin()
	if err != nil {
		log.Println(err)
	}
	stmt, err := tx.Prepare("insert into worklog(hours, desc, projcode, starttime, endtime, date, notes) values(?, ?, ?, ?, ?, ?, ?)")

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(entry.entry.hours,
		entry.entry.desc, entry.entry.projCode, entry.entry.startTime,
		entry.entry.endTime, entry.entry.date, entry.entry.notes)
	if err != nil {
		log.Println(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Database) DeleteEntry(e int) error {
	sqlstmt := `delete from worklog where id = ?;`
	tx, err := d.db.Begin()
	if err != nil {
		log.Println(err)
	}
	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		log.Println(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(e)
	if err != nil {
		log.Println(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Database) ModifyEntry(e EntryRow) error {
	// TODO: Could optimise to only update what is changed
	sqlstmt := `Update worklog set desc = ?, 
				hours = ?, 
				projcode = ?, 
				date = ?, 
				starttime = ?, 
				endtime = ?, 
				notes = ? 
			where id = ?;`
	tx, err := d.db.Begin()
	if err != nil {
		log.Println(err)
	}
	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		log.Println(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(e.entry.desc, e.entry.hours,
		e.entry.projCode, e.entry.date, e.entry.startTime,
		e.entry.endTime, e.entry.notes, e.entryId)
	if err != nil {
		log.Println(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (d *Database) QuerySummary(m *model) ([]EntryRow, error) {
	// Use this to get a summary of the past week of entries
	// Or get a summary of the
	// Potentially later modify the time length being requested.
	var (
		rows *sql.Rows
		err  error
		ents []EntryRow
	)
	//fmt.Println(m.currentDate.String())
	m.currentDate = time.Now()
	startDate := m.startDate.Format("2006-01-02")
	endDate := m.endDate.AddDate(0, 0, 1).Format("2006-01-02")
	//fmt.Println(fmt.Sprintf("select date, id, projcode, hours, desc from worklog where date between date(%s) and date(%s)", startDate, endDate))

	rows, err = d.db.Query("select date, id, projcode, hours, desc from worklog where date between date(?) and date(?) order by date desc", startDate, endDate)
	if err != nil {
		return []EntryRow{}, err
	}
	defer rows.Close()
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.entry.date, &ent.entryId, &ent.entry.projCode, &ent.entry.hours, &ent.entry.desc)
		if err != nil {
			return []EntryRow{}, err
		}
		//fmt.Println(ent.entryId, ent.entry.projCode)
		ents = append(ents, ent)
	}
	err = rows.Err()
	if err != nil {
		return []EntryRow{}, err
	}
	return ents, nil
}

// TODO: fix this to make list construction work better
func (d *Database) QueryEntries(m *model) ([]EntryRow, error) {
	var (
		rows  *sql.Rows
		err   error
		notes sql.NullString
	)
	if m.id == 0 {
		rows, err = d.db.Query("select date, id, projcode, hours, desc, notes from worklog order by id desc limit 10")
	} else {
		rows, err = d.db.Query("select date, id, projcode, hours, desc, notes from worklog order by id desc limit 10 offset ?", m.maxId-m.id+1)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	ents := []EntryRow{}
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.entry.date, &ent.entryId, &ent.entry.projCode, &ent.entry.hours, &ent.entry.desc, &notes)
		if err != nil {
			log.Fatal(err)
		}
		// Check if newColumn is valid (non-NULL) and print it
		ent.entry.notes = ""
		if notes.Valid {
			ent.entry.notes = notes.String
			//log.Println(notes)
		}
		ents = append(ents, ent)
		if m.id == 0 {
			m.maxId = ent.entryId
		}
		m.id = ent.entryId
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return ents, nil
}

func (d *Database) QueryAndExport() error {
	var (
		rows  *sql.Rows
		err   error
		notes sql.NullString
	)
	prevDate := time.Time{}
	f, err := os.OpenFile("export.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0755)
	if err != nil {
		return err
	}
	rows, err = d.db.Query("select date, id, projcode, hours, desc, notes from worklog order by date ASC")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer rows.Close()
	bufferSize := 64 * 1024 // 64 KB
	w := bufio.NewWriterSize(f, bufferSize)
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.entry.date, &ent.entryId, &ent.entry.projCode, &ent.entry.hours, &ent.entry.desc, &notes)
		if err != nil {
			log.Fatal(err)
			return err
		}
		ent.entry.notes = ""
		if notes.Valid {
			ent.entry.notes = notes.String
			//log.Println(notes)
		}
		if prevDate != ent.entry.date {
			prevDate = ent.entry.date
			_, err = w.WriteString(fmt.Sprintf("%s\n", ent.entry.date.Format("2006-01-02")))
			if err != nil {
				log.Println(err)
			}
		}
		_, err = w.WriteString(fmt.Sprintf("\t%02d:%02d:%02d %s\n\t\t%s\n", int(ent.entry.hours.Hours()), int(ent.entry.hours.Minutes())%60, int(ent.entry.hours.Seconds())%60, ent.entry.projCode, ent.entry.desc))
		if err != nil {
			log.Println(err)
		}
		w.Flush()
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
		return err
	}
	f.Close()
	return nil
}

func (d *Database) QueryEntry(e EntryRow) (EntryRow, error) {
	var (
		rows  *sql.Rows
		err   error
		notes sql.NullString
	)
	rows, _ = d.db.Query("select date, id, projcode, hours, desc, starttime, endtime, notes from worklog where id = %d", e.entryId)
	defer rows.Close()
	var ent EntryRow
	for rows.Next() {
		err = rows.Scan(&ent.entry.date, &ent.entryId, &ent.entry.projCode, &ent.entry.hours, &ent.entry.desc, &ent.entry.startTime, &ent.entry.endTime, &notes)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(ent.entryId, ent.entry.projCode)
	}
	// Check if newColumn is valid (non-NULL) and print it
	ent.entry.notes = ""
	if notes.Valid {
		ent.entry.notes = notes.String
		//log.Println(notes)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return ent, nil
}

func (d *Database) CreateDatabase() error {

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS worklog (
		id INTEGER NOT NULL PRIMARY KEY, 
		hours TIME NOT NULL, 
		desc TEXT, 
		starttime TIME NOT NULL, 
		endtime TIME, 
		projcode TEXT NOT NULL, 
		date DATE NOT NULL,
		notes TEXT
	);`
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
	d.CreateDatabase()

	d.CreateEventDatabase()
	d.AlterTable()
	//defer row.Close()
	return nil
}

func (d *Database) CloseDatabase() {
	d.db.Close()
}

func (e *EntryRow) FillData(m *model) error {
	timeFmt := "15:04"
	dateFmt := "02/01/2006"
	var err error
	e.entry.hours, err = time.ParseDuration(m.inputs[hours].Value())
	if err != nil && m.inputs[hours].Value() != "" {
		log.Println(err)
		return err
	}
	e.entry.startTime, err = time.Parse(timeFmt, m.inputs[startTime].Value())
	if err != nil && !e.entry.startTime.IsZero() {
		log.Println(err)
		return err
	}
	e.entry.endTime, err = time.Parse(timeFmt, m.inputs[endTime].Value())
	if err != nil && !e.entry.endTime.IsZero() {
		log.Println(err)
		return err
	}
	e.entry.date, err = time.Parse(dateFmt, m.inputs[date].Value())
	if err != nil {
		log.Println(err)
		return err
	}
	e.entry.projCode = m.inputs[code].Value()
	e.entry.desc = m.inputs[desc].Value()
	e.entry.notes = m.textarea.Value()

	if !e.entry.startTime.IsZero() && !e.entry.endTime.IsZero() {
		e.entry.hours = time.Duration(e.entry.endTime.Sub(e.entry.startTime))
	}

	// Now do some validation checks on projcode and hours to make sure they exist.
	if e.entry.hours.Minutes() == 0 || e.entry.projCode == "" {
		log.Println(e.entry.hours.Minutes(), e.entry.projCode)
		return fmt.Errorf("empty hours or projcode, please check inputs")
	}
	return nil
}

func (e *EntryRow) ModFillData(m *model) error {
	timeFmt := "15:04"
	dateFmt := "02/01/2006"
	var err error
	e.entry.hours, err = time.ParseDuration(m.modInputs[hours].Value())
	if err != nil && m.modInputs[hours].Value() != "" {
		log.Println(err)
		return err
	}
	e.entry.startTime, err = time.Parse(timeFmt, m.modInputs[startTime].Value())
	if err != nil && !e.entry.startTime.IsZero() {
		log.Println(err)
		return err
	}
	e.entry.endTime, err = time.Parse(timeFmt, m.modInputs[endTime].Value())
	if err != nil && !e.entry.endTime.IsZero() {
		log.Println(err)
		return err
	}
	e.entry.date, err = time.Parse(dateFmt, m.modInputs[date].Value())
	if err != nil {
		log.Println(err)
		return err
	}
	e.entry.projCode = m.modInputs[code].Value()
	e.entry.desc = m.modInputs[desc].Value()
	e.entry.notes = m.modtextarea.Value()

	if !e.entry.startTime.IsZero() && !e.entry.endTime.IsZero() {
		e.entry.hours = time.Duration(e.entry.endTime.Sub(e.entry.startTime))
	}

	// Now do some validation checks on projcode and hours to make sure they exist.
	if e.entry.hours.Minutes() == 0 || e.entry.projCode == "" {
		log.Println(e.entry.hours.Minutes(), e.entry.projCode)
		return fmt.Errorf("empty hours or projcode, please check inputs")
	}
	return nil
}

// TODO: Write some functions for creating, reading and writing from the Proj/Event_Id table.
// read from startup

func (d *Database) CreateEventDatabase() error {

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS projeventlink 
		(id INTEGER PRIMARY KEY, 
		projcode TEXT NOT NULL, 
		eventid INTEGER NOT NULL,
		UNIQUE(projcode)
		);`
	_, err := d.db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		//return fmt.Errorf("db stmt fail %q: %s", err, sqlStmt)
	}
	log.Println("Table 'users' created successfully (or already exists)")
	return nil
}

// Check if table has the notes column and add if not
func (d *Database) AlterTable() error {
	query := "PRAGMA table_info(worklog);"
	rows, err := d.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var exists bool
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString

		// Scan each row from the table schema
		err = rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			return err
		}

		// Check if the column exists
		if name == "notes" {
			exists = true
			break
		}
	}

	if !exists {
		// Add the column since it doesn't exist
		query := "ALTER TABLE worklog ADD COLUMN notes TEXT;"
		_, err = d.db.Exec(query)
		if err != nil {
			return err
		}
		log.Println("Column notes added to table worklog")
	} else {
		log.Println("Column notes already exists in table worklog")
	}
	return nil
}

// This should be the only function required to read links
func (d *Database) QueryLinks() (map[string]int, error) {
	records := make(map[string]int)

	// Query the table for all rows
	rows, err := d.db.Query("SELECT projcode, eventid FROM projeventlink")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Loop through the result set and add the values to the map
	for rows.Next() {
		var projCode string
		var eventID int
		err = rows.Scan(&projCode, &eventID) // Scan each row into the variables
		if err != nil {
			log.Println(err)
			return map[string]int{}, err
		}
		// Add the result to the map
		records[projCode] = eventID
	}

	// Check for any error that occurred during the iteration
	if err = rows.Err(); err != nil {
		log.Println(err)
		return map[string]int{}, err
	}

	// Print the map to verify the data
	log.Println("Records from the database:", records)
	return records, nil
}

func (d *Database) SaveLink(proj string, id int) error {
	tx, err := d.db.Begin()
	if err != nil {
		log.Println(err)
	}
	stmt, err := tx.Prepare("INSERT INTO projeventlink(projcode, eventid) values(?, ?)")

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(proj, id)
	if err != nil {
		tx.Rollback()
		//log.Println(err)
		return err
	}
	// Below code for if we decide to batch the write
	// for code, value := range linkmap {
	// 	_, err = stmt.Exec(code, value)
	// 	if err != nil {
	// 		tx.Rollback()
	// 		log.Fatal(err)
	// 		return err
	// 	}
	// }
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Database) DeleteLink(projCode string) error {
	// Prepare the DELETE SQL statement
	stmt, err := d.db.Prepare("DELETE FROM projeventlink WHERE projcode = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Execute the statement with the provided projCode and eventID
	_, err = stmt.Exec(projCode)
	if err != nil {
		return err
	}
	// Log if anything was deleted.
	// rows, err := res.RowsAffected()
	// if err != nil {
	// 	return err
	// }
	// if rows == 0 {
	// 	log.Printf("No link found for projcode '%s'.\n", projCode)
	// } else {
	// 	log.Printf("Deleted %d link(s) for projcode '%s'.\n", rows, projCode)
	// }

	return nil
}
