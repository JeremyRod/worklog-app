package internal

import (
	"testing"
	"time"
)

var db Database = Database{Db: nil}

func TestOpenDatabase(t *testing.T) {
	err := db.OpenDatabase(t)
	if err != nil {
		t.Fatalf(`OpenDatabase() = %v`, err)
	}
}

func TestSaveDatabase(t *testing.T) {
	err := db.OpenDatabase(t)
	if err != nil {
		t.Fatalf(`OpenDatabase() = %v`, err)
	}
	hours, _ := time.ParseDuration("01h30m")
	row := EntryRow{Entry: Entry{
		Hours:     hours,
		Desc:      "Database test input",
		Notes:     "",
		StartTime: time.Date(2009, 11, 30, 12, 30, 0, 0, time.Local),
		EndTime:   time.Date(2009, 11, 30, 13, 30, 0, 0, time.Local),
		Date:      time.Date(2009, 11, 30, 11, 30, 0, 0, time.Local),
	}}
	err = db.SaveEntry(row)
	if err != nil {
		t.Fatalf(`SaveEntry() = %v`, err)
	}
}

func TestQueryDatabase(t *testing.T) {
	err := db.OpenDatabase(t)
	hours, _ := time.ParseDuration("01h30m")
	id := 0
	max := 0
	row := EntryRow{Entry: Entry{
		Hours:     hours,
		Desc:      "Database test input",
		Notes:     "",
		StartTime: time.Date(2009, 11, 30, 12, 30, 0, 0, time.Local),
		EndTime:   time.Date(2009, 11, 30, 13, 30, 0, 0, time.Local),
		Date:      time.Date(2009, 11, 30, 11, 30, 0, 0, time.Local),
	}}
	if err != nil {
		t.Fatalf(`OpenDatabase() = %v`, err)
	}
	entry, err := db.QueryEntries(&id, &max)
	if err != nil {
		t.Fatalf(`QueryEntries() = %v`, err)
	}
	for _, v := range entry {
		if v.Entry.Hours != row.Entry.Hours {
			t.Fatalf(`Hours Mismatch = %v`, err)
		}
		if v.Entry.Date.Format("2006-01-02 15:04:05-07:00") != row.Entry.Date.Format("2006-01-02 15:04:05-07:00") {
			t.Log(v.Entry.Date, row.Entry.Date)
			t.Fatalf(`Date Mismatch = %v`, err)
		}
		if v.Entry.Desc != row.Entry.Desc {
			t.Fatalf(`Desc Mismatch = %v`, err)
		}
		if v.Entry.EndTime != row.Entry.EndTime {
			t.Fatalf(`Endtime Mismatch = %v`, err)
		}
		if v.Entry.StartTime != row.Entry.StartTime {
			t.Fatalf(`Starttime Mismatch = %v`, err)
		}
		if v.Entry.Notes != row.Entry.Notes {
			t.Fatalf(`Notes Mismatch = %v`, err)
		}
	}
}

func TestModifyEntry(t *testing.T) {
	err := db.OpenDatabase(t)
	hours, _ := time.ParseDuration("01h30m")
	id := 0
	max := 0
	row := EntryRow{Entry: Entry{
		Hours:     hours,
		Desc:      "Database test input Modification",
		Notes:     "We have modified",
		StartTime: time.Date(2010, 11, 30, 12, 30, 0, 0, time.Local),
		EndTime:   time.Date(2010, 11, 30, 13, 30, 0, 0, time.Local),
		Date:      time.Date(2010, 11, 30, 11, 30, 0, 0, time.Local),
	}}
	if err != nil {
		t.Fatalf(`OpenDatabase() = %v`, err)
	}
	entry, err := db.QueryEntries(&id, &max)
	if err != nil {
		t.Fatalf(`QueryEntries() = %v`, err)
	}
	for range entry {
		db.ModifyEntry(row)
	}
	entry, err = db.QueryEntries(&id, &max)
	if err != nil {
		t.Fatalf(`QueryEntries() = %v`, err)
	}
	for _, v := range entry {
		if v.Entry.Hours != row.Entry.Hours {
			t.Fatalf(`Hours Mismatch = %v`, err)
		}
		if v.Entry.Date.Format("2006-01-02 15:04:05-07:00") != row.Entry.Date.Format("2006-01-02 15:04:05-07:00") {
			t.Log(v.Entry.Date, row.Entry.Date)
			t.Fatalf(`Date Mismatch = %v`, err)
		}
		if v.Entry.Desc != row.Entry.Desc {
			t.Fatalf(`Desc Mismatch = %v`, err)
		}
		if v.Entry.EndTime != row.Entry.EndTime {
			t.Fatalf(`Endtime Mismatch = %v`, err)
		}
		if v.Entry.StartTime != row.Entry.StartTime {
			t.Fatalf(`Starttime Mismatch = %v`, err)
		}
		if v.Entry.Notes != row.Entry.Notes {
			t.Fatalf(`Notes Mismatch = %v`, err)
		}
	}
}
func TestDeleteEntry(t *testing.T) {
	err := db.OpenDatabase(t)
	if err != nil {
		t.Fatalf(`OpenDatabase() = %v`, err)
	}
	id := 0
	max := 0
	entry, err := db.QueryEntries(&id, &max)
	if err != nil {
		t.Fatalf(`QueryEntries() = %v`, err)
	}
	for _, v := range entry {
		if err := db.DeleteEntry(v.EntryId); err != nil {
			t.Fatalf(`DeleteEntry() = %v`, err)
		}
	}
}
func TestLinkAct(t *testing.T) {
	proj := "SRO"
	id := 25
	err := db.OpenDatabase(t)
	if err != nil {
		t.Fatalf(`OpenDatabase() = %v`, err)
	}
	db.SaveLink(proj, id)
	if err != nil {
		t.Fatalf(`SaveLink() = %v`, err)
	}
	db.SaveAct(proj, id)
	if err != nil {
		t.Fatalf(`SaveAct() = %v`, err)
	}
	links, acts, err := db.QueryLinks()
	if err != nil {
		t.Fatalf(`QueryLinks() = %v`, err)
	}
	for p, i := range links {
		if p != proj {
			t.Fatal(`Link ProjCode Mismatch`)
		}
		if i != id {
			t.Fatal(`Link Id Mismatch`)
		}
	}
	for p, i := range acts {
		if p != proj {
			t.Fatal(`Act ProjCode Mismatch`)
		}
		if i != id {
			t.Fatal(`Act Id Mismatch`)
		}
	}
	err = db.DeleteLink(proj)
	if err != nil {
		t.Fatalf(`DeleteLink() = %v`, err)
	}
}
