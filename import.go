package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// For the moment the import function will only look for a file
func ImportWorklog() error {
	file, err := os.Open("worklog.txt")
	if err != nil {
		return err
	}
	rd := bufio.NewReader(file)
	e := EntryRow{}
	for {
		bytes, _ := rd.Peek(2)
		str, err := rd.ReadString('\n')
		if err != nil {
			switch err {
			case io.EOF:
				return nil
			default:
				return err
			}
		}

		str = strings.Replace(str, "\r\n", "", -1)
		str = strings.Replace(str, "\t", "", -1)
		//fmt.Println(str)
		if bytes[0] == '\r' && bytes[1] == '\n' {
			// Shows a new empty line.
		} else if bytes[0] == '\t' && bytes[1] == '\t' {
			// Data lines should be double indented (or more) with tabs
			check, err := rd.Peek(1)
			if check[0] == '\r' || check[0] == '\n' || err != nil {
				continue
			}
			e.entry.desc += str
			e.entry.desc += "\n"
		} else if bytes[0] == '\t' && bytes[1] != '\t' {
			// Time, proj code and tags on single indent lines.
			// Check that this isnt a line with nothing but a tab, happens
			if bytes[1] == '\r' || bytes[1] == '\n' {
				continue
			}
			// Finish old entry and submit.
			strArr := strings.Split(str, " ")

			if !e.entry.startTime.IsZero() {
				e.entry.endTime, err = time.Parse("15:04", strings.Trim(strArr[0], "\t"))
				if err != nil {
					return fmt.Errorf("endtime parse fail")
				}
				e.entry.hours = time.Duration(e.entry.endTime.Sub(e.entry.startTime))

				db.SaveEntry(e)
			}
			// Start new entry
			e.entry.desc = ""
			e.entry.projCode = strArr[1]
			e.entry.startTime, err = time.Parse("15:04", strings.Trim(strArr[0], "\t"))
			if err != nil {
				return fmt.Errorf("startTime parse fail")
			}

		} else {
			// None indented are new dates.
			// If we hit a new date, the last potential entry should probably just be cleared.
			e = EntryRow{}
			e.entry.date, err = time.Parse("2006-01-02", str)
			if err != nil {
				return fmt.Errorf("date parse fail")
			}
		}
	}
	return nil
}
