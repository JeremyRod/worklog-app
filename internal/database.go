package internal

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	_ "github.com/mattn/go-sqlite3"
)

const (
	Date = iota
	Code
	Desc
	StartTime
	EndTime
	Hours
	Submit
	Deleted
	Imp
	Unlink
)

type Database struct {
	Db *sql.DB
}

type Entry struct {
	Hours     time.Duration
	ProjCode  string
	Desc      string
	StartTime time.Time
	EndTime   time.Time
	Date      time.Time
	Notes     string
}

type EntryRow struct {
	Entry   Entry
	EntryId int
}

// FIXME: Fix the formatting here
func (e EntryRow) Title() string {
	date := e.Entry.Date.Format("02/01/2006")
	time := fmt.Sprintf("%d:%02d", int(e.Entry.Hours.Hours()), int(e.Entry.Hours.Minutes())%60)
	return fmt.Sprintf("Date: %v Project: %s Hours: %s", date, e.Entry.ProjCode, time)
}
func (e EntryRow) Description() string { return e.Entry.Desc }
func (e EntryRow) FilterValue() string { return e.Entry.ProjCode }

func (d *Database) SaveEntry(entry EntryRow) error {
	tx, err := d.Db.Begin()
	if err != nil {
		logger.Println(err)
	}
	stmt, err := tx.Prepare("insert into worklog(hours, desc, projcode, starttime, endtime, date, notes) values(?, ?, ?, ?, ?, ?, ?)")

	if err != nil {
		logger.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(entry.Entry.Hours,
		entry.Entry.Desc, entry.Entry.ProjCode, entry.Entry.StartTime,
		entry.Entry.EndTime, entry.Entry.Date, entry.Entry.Notes)
	if err != nil {
		logger.Println(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal(err)
	}
	return nil
}

func (d *Database) DeleteEntry(e int) error {
	sqlstmt := `delete from worklog where id = ?;`
	tx, err := d.Db.Begin()
	if err != nil {
		logger.Println(err)
	}
	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		logger.Println(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(e)
	if err != nil {
		logger.Println(err)
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal(err)
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
	tx, err := d.Db.Begin()
	if err != nil {
		logger.Println(err)
	}
	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		logger.Println(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(e.Entry.Desc, e.Entry.Hours,
		e.Entry.ProjCode, e.Entry.Date, e.Entry.StartTime,
		e.Entry.EndTime, e.Entry.Notes, e.EntryId)
	if err != nil {
		logger.Println(err)
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal(err)
	}

	return nil
}

func (d *Database) QuerySummary(start, end *time.Time) ([]EntryRow, error) {
	// Use this to get a summary of the past week of entries
	// Or get a summary of the
	// Potentially later modify the time length being requested.
	var (
		rows *sql.Rows
		err  error
		ents []EntryRow
	)
	//fmt.Println(m.currentDate.String())
	startDate := start.Format("2006-01-02")
	endDate := end.AddDate(0, 0, 1).Format("2006-01-02")
	//fmt.Println(fmt.Sprintf("select date, id, projcode, hours, desc from worklog where date between date(%s) and date(%s)", startDate, endDate))

	rows, err = d.Db.Query("select date, id, projcode, hours, desc from worklog where date between date(?) and date(?) order by date desc", startDate, endDate)
	if err != nil {
		return []EntryRow{}, err
	}
	defer rows.Close()
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.Entry.Date, &ent.EntryId, &ent.Entry.ProjCode, &ent.Entry.Hours, &ent.Entry.Desc)
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
func (d *Database) QueryEntries(id, maxId *int) ([]EntryRow, error) {
	var (
		rows               *sql.Rows
		err                error
		notes              sql.NullString
		startTime, endTime string
	)
	if *id == 0 {
		rows, err = d.Db.Query("select date, id, projcode, hours, desc, notes, starttime, endtime from worklog order by id desc limit 10")
	} else {
		rows, err = d.Db.Query("select date, id, projcode, hours, desc, notes, starttime, endtime from worklog order by id desc limit 10 offset ?", *maxId-*id+1)
	}
	if err != nil {
		logger.Fatal(err)
	}
	defer rows.Close()
	ents := []EntryRow{}
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.Entry.Date, &ent.EntryId, &ent.Entry.ProjCode, &ent.Entry.Hours, &ent.Entry.Desc, &notes, &startTime, &endTime)
		if err != nil {
			logger.Fatal(err)
		}
		// Check if newColumn is valid (non-NULL) and print it
		ent.Entry.Notes = ""
		if notes.Valid {
			ent.Entry.Notes = notes.String
			//log.Println(notes)
		}
		ent.Entry.StartTime, err = time.Parse("2006-01-02 15:04:05-07:00", startTime)
		if err != nil {
			logger.Println(err)
		}
		ent.Entry.EndTime, err = time.Parse("2006-01-02 15:04:05-07:00", endTime)
		if err != nil {
			logger.Println(err)
		}
		ents = append(ents, ent)
		if *id == 0 {
			*maxId = ent.EntryId
		}
		*id = ent.EntryId
	}
	err = rows.Err()
	if err != nil {
		logger.Fatal(err)
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
	rows, err = d.Db.Query("select date, id, projcode, hours, desc, notes from worklog order by date ASC")
	if err != nil {
		logger.Fatal(err)
		return err
	}
	defer rows.Close()
	bufferSize := 64 * 1024 // 64 KB
	w := bufio.NewWriterSize(f, bufferSize)
	for rows.Next() {
		ent := EntryRow{}
		err = rows.Scan(&ent.Entry.Date, &ent.EntryId, &ent.Entry.ProjCode, &ent.Entry.Hours, &ent.Entry.Desc, &notes)
		if err != nil {
			logger.Fatal(err)
			return err
		}
		ent.Entry.Notes = ""
		if notes.Valid {
			ent.Entry.Notes = notes.String
			//log.Println(notes)
		}
		if prevDate != ent.Entry.Date {
			prevDate = ent.Entry.Date
			_, err = w.WriteString(fmt.Sprintf("%s\n", ent.Entry.Date.Format("2006-01-02")))
			if err != nil {
				logger.Println(err)
			}
		}
		_, err = w.WriteString(fmt.Sprintf("\t%02d:%02d:%02d %s\n\t\t%s\n", int(ent.Entry.Hours.Hours()), int(ent.Entry.Hours.Minutes())%60, int(ent.Entry.Hours.Seconds())%60, ent.Entry.ProjCode, ent.Entry.Desc))
		if err != nil {
			logger.Println(err)
		}
		w.Flush()
	}
	err = rows.Err()
	if err != nil {
		logger.Fatal(err)
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
	rows, _ = d.Db.Query("select date, id, projcode, hours, desc, starttime, endtime, notes from worklog where id = %d", e.EntryId)
	defer rows.Close()
	var ent EntryRow
	for rows.Next() {
		err = rows.Scan(&ent.Entry.Date, &ent.EntryId, &ent.Entry.ProjCode, &ent.Entry.Hours, &ent.Entry.Desc, &ent.Entry.StartTime, &ent.Entry.EndTime, &notes)
		if err != nil {
			logger.Fatal(err)
		}
		//fmt.Println(ent.entryId, ent.entry.projCode)
	}
	// Check if newColumn is valid (non-NULL) and print it
	ent.Entry.Notes = ""
	if notes.Valid {
		ent.Entry.Notes = notes.String
		//log.Println(notes)
	}
	err = rows.Err()
	if err != nil {
		logger.Fatal(err)
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
	_, err := d.Db.Exec(sqlStmt)
	if err != nil {
		logger.Printf("%q: %s\n", err, sqlStmt)
		return fmt.Errorf("db stmt fail %q: %s", err, sqlStmt)
	}
	return nil
}

// TODO: Seed database for better tests.
func (d *Database) SeedDatabase() error {
	tx, err := d.Db.Begin()
	if err != nil {
		logger.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into worklog(id, name) values(?, ?)")
	if err != nil {
		logger.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("こんにちは世界%03d", i))
		if err != nil {
			logger.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal(err)
	}
	return nil
}

func (d *Database) OpenDatabase(t *testing.T) error {
	var (
		db  *sql.DB
		err error
	)
	if t != nil {
		db, err = sql.Open("sqlite3", "./test.db")
	} else {
		db, err = sql.Open("sqlite3", "./worklog.db")
	}
	if err != nil {
		return errors.New("database broke")
	}
	d.Db = db
	d.CreateDatabase()

	d.CreateEventDatabase()
	d.AlterTable()
	d.AlterProjTable()
	//defer row.Close()
	return nil
}

func (d *Database) CloseDatabase() {
	d.Db.Close()
}

func (e *EntryRow) FillData(inputs []textinput.Model, textarea *textarea.Model) error {
	timeFmt := "15:04"
	dateFmt := "02/01/2006"
	var err error
	e.Entry.Hours, err = time.ParseDuration(inputs[Hours].Value())
	if err != nil && inputs[Hours].Value() != "" {
		logger.Println(err)
		return err
	}
	e.Entry.StartTime, err = time.Parse(timeFmt, inputs[StartTime].Value())
	if err != nil && !e.Entry.StartTime.IsZero() {
		logger.Println(err)
		return err
	}
	e.Entry.EndTime, err = time.Parse(timeFmt, inputs[EndTime].Value())
	if err != nil && !e.Entry.EndTime.IsZero() {
		logger.Println(err)
		return err
	}
	e.Entry.Date, err = time.Parse(dateFmt, inputs[Date].Value())
	if err != nil {
		logger.Println(err)
		return err
	}
	e.Entry.ProjCode = inputs[Code].Value()
	e.Entry.Desc = inputs[Desc].Value()
	e.Entry.Notes = textarea.Value()

	if !e.Entry.StartTime.IsZero() && !e.Entry.EndTime.IsZero() {
		e.Entry.Hours = time.Duration(e.Entry.EndTime.Sub(e.Entry.StartTime))
	}

	// Now do some validation checks on projcode and hours to make sure they exist.
	if e.Entry.Hours.Minutes() == 0 || e.Entry.ProjCode == "" {
		logger.Println(e.Entry.Hours.Minutes(), e.Entry.ProjCode)
		return fmt.Errorf("empty hours or projcode, please check inputs")
	}
	return nil
}

func (e *EntryRow) ModFillData(inputs []textinput.Model, textarea *textarea.Model) error {
	timeFmt := "15:04"
	dateFmt := "02/01/2006"
	var err error
	e.Entry.Hours, err = time.ParseDuration(inputs[Hours].Value())
	if err != nil && inputs[Hours].Value() != "" {
		logger.Println(err)
		return err
	}
	e.Entry.StartTime, err = time.Parse(timeFmt, inputs[StartTime].Value())
	if err != nil && !e.Entry.StartTime.IsZero() {
		logger.Println(err)
		return err
	}
	e.Entry.EndTime, err = time.Parse(timeFmt, inputs[EndTime].Value())
	if err != nil && !e.Entry.EndTime.IsZero() {
		logger.Println(err)
		return err
	}
	e.Entry.Date, err = time.Parse(dateFmt, inputs[Date].Value())
	if err != nil {
		logger.Println(err)
		return err
	}
	e.Entry.ProjCode = inputs[Code].Value()
	e.Entry.Desc = inputs[Desc].Value()
	e.Entry.Notes = textarea.Value()

	if !e.Entry.StartTime.IsZero() && !e.Entry.EndTime.IsZero() {
		e.Entry.Hours = time.Duration(e.Entry.EndTime.Sub(e.Entry.StartTime))
	}

	// Now do some validation checks on projcode and hours to make sure they exist.
	if e.Entry.Hours.Minutes() == 0 || e.Entry.ProjCode == "" {
		logger.Println(e.Entry.Hours.Minutes(), e.Entry.ProjCode)
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
		activity INTEGER,
		updateflag BOOLEAN DEFAULT FALSE,
		UNIQUE(projcode)
		);`
	_, err := d.Db.Exec(sqlStmt)
	if err != nil {
		logger.Fatalf("%q: %s\n", err, sqlStmt)
		//return fmt.Errorf("db stmt fail %q: %s", err, sqlStmt)
	}
	logger.Println("Table 'projeventlink' created successfully (or already exists)")
	return nil
}

// Check if table has the notes column and add if not
func (d *Database) AlterProjTable() error {
	query := "PRAGMA table_info(projeventlink);"
	rows, err := d.Db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var actExist, updateFlagExist bool
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

		// Check if the columns exist
		if name == "activity" {
			actExist = true
		}
		if name == "updateflag" {
			updateFlagExist = true
		}
	}

	if !actExist {
		// Add the column since it doesn't exist
		query := "ALTER TABLE projeventlink ADD COLUMN activity INTEGER;"
		_, err = d.Db.Exec(query)
		if err != nil {
			return err
		}
		logger.Println("Column activity added to table projeventlink")
	} else {
		logger.Println("Column activity already exists in table projeventlink")
	}

	if !updateFlagExist {
		// Add the column since it doesn't exist
		query := "ALTER TABLE projeventlink ADD COLUMN updateflag BOOLEAN DEFAULT FALSE;"
		_, err = d.Db.Exec(query)
		if err != nil {
			return err
		}
		logger.Println("Column updateflag added to table projeventlink")
	} else {
		logger.Println("Column updateflag already exists in table projeventlink")
	}
	return nil
}

// Check if table has the notes column and add if not
func (d *Database) AlterTable() error {
	query := "PRAGMA table_info(worklog);"
	rows, err := d.Db.Query(query)
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
		_, err = d.Db.Exec(query)
		if err != nil {
			return err
		}
		logger.Println("Column notes added to table worklog")
	} else {
		logger.Println("Column notes already exists in table worklog")
	}
	return nil
}

// This should be the only function required to read links
func (d *Database) QueryLinks() (map[string]int, map[string]int, map[string]bool, error) {
	records := make(map[string]int)
	actIDs := make(map[string]int)
	updateFlags := make(map[string]bool)

	// Query the table for all rows
	rows, err := d.Db.Query("SELECT projcode, eventid, activity, updateflag FROM projeventlink")
	if err != nil {
		logger.Fatal(err)
	}
	defer rows.Close()

	// Loop through the result set and add the values to the map
	for rows.Next() {
		var projCode string
		var eventID int
		var activityID sql.NullInt32
		var updateFlag sql.NullBool
		err = rows.Scan(&projCode, &eventID, &activityID, &updateFlag) // Scan each row into the variables
		if err != nil {
			logger.Println(err)
			return map[string]int{}, map[string]int{}, map[string]bool{}, err
		}
		// Add the result to the map
		records[projCode] = eventID
		actIDs[projCode] = -1
		if activityID.Valid {
			actIDs[projCode] = int(activityID.Int32)
		}
		updateFlags[projCode] = false
		if updateFlag.Valid {
			updateFlags[projCode] = updateFlag.Bool
		}
	}

	// Check for any error that occurred during the iteration
	if err = rows.Err(); err != nil {
		logger.Println(err)
		return map[string]int{}, map[string]int{}, map[string]bool{}, err
	}

	// Print the map to verify the data
	logger.Println("Records from the database:", records)
	logger.Println("ActIds from the database:", actIDs)
	logger.Println("Update flags from the database:", updateFlags)
	return records, actIDs, updateFlags, nil
}

func (d *Database) SaveLink(proj string, id int) error {
	tx, err := d.Db.Begin()
	if err != nil {
		logger.Println(err)
	}
	stmt, err := tx.Prepare("INSERT INTO projeventlink(projcode, eventid) values(?, ?)")

	if err != nil {
		logger.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(proj, id)
	if err != nil {
		tx.Rollback()
		//log.Println(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal(err)
	}
	return nil
}

func (d *Database) SaveAct(proj string, id int) error {
	tx, err := d.Db.Begin()
	if err != nil {
		logger.Println(err)
	}
	stmt, err := tx.Prepare("UPDATE projeventlink SET activity = ? WHERE projcode = ?")

	if err != nil {
		logger.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(id, proj)
	if err != nil {
		tx.Rollback()
		//log.Println(err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal(err)
	}
	return nil
}

func (d *Database) DeleteLink(projCode string) error {
	// Prepare the DELETE SQL statement
	stmt, err := d.Db.Prepare("DELETE FROM projeventlink WHERE projcode = ?")
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

func (d *Database) SetUpdateFlag() error {
	sqlstmt := `UPDATE projeventlink SET updateflag = TRUE;`
	tx, err := d.Db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (d *Database) SetUpdateFlagFalse(projCodes []string) error {
	if len(projCodes) == 0 {
		return nil
	}

	// Create placeholders for the IN clause
	placeholders := make([]string, len(projCodes))
	args := make([]interface{}, len(projCodes))
	for i := range projCodes {
		placeholders[i] = "?"
		args[i] = projCodes[i]
	}

	sqlstmt := fmt.Sprintf(`UPDATE projeventlink SET updateflag = FALSE WHERE projcode IN (%s);`, strings.Join(placeholders, ","))
	tx, err := d.Db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(sqlstmt)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
