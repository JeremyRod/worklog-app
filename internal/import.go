package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// For the moment the import function will only look for a file
func ImportWorklog(db *Database) (int, error) {
	file, err := os.Open("worklog.txt")
	if err != nil {
		return -1, err
	}
	rd := bufio.NewReaderSize(file, 32*1024)
	// fmt.Println(rd.Size())
	e := EntryRow{}
	lineNum := 0
	for {
		// bytes, _ := rd.Peek(2)
		str, err := rd.ReadString('\n')
		if err != nil {
			switch err {
			case io.EOF:
				return -1, nil
			default:
				return -1, err
			}
		}
		lineNum++
		strBytes := []byte(str)
		//fmt.Println(strBytes)
		str = strings.Replace(str, "\r\n", "", -1)
		str = strings.Replace(str, "\t", "", -1)
		if strBytes[0] == ' ' || strBytes[1] == ' ' {
			var found bool
			str, found = strings.CutPrefix(str, " \t")
			if !found {
				str, _ = strings.CutPrefix(str, "\t ")
			}
		}

		if strBytes[0] == '\r' && strBytes[1] == '\n' {
			// Shows a new empty line.
		} else if strBytes[0] == '\t' && strBytes[1] == '\t' {
			// Data lines should be double indented (or more) with tabs
			check, err := rd.Peek(1)
			if check[0] == '\r' || check[0] == '\n' || err != nil {
				continue
			}
			e.Entry.Desc += str
			e.Entry.Desc += "\n"
		} else if strBytes[0] == '\t' && strBytes[1] != '\t' {
			// Time, proj code and tags on single indent lines.
			// Check that this isnt a line with nothing but a tab, happens
			if strBytes[1] == '\r' || strBytes[1] == '\n' {
				continue
			}
			// Finish old entry and submit.
			strArr := strings.Split(str, " ")

			if !e.Entry.StartTime.IsZero() {
				e.Entry.EndTime, err = time.Parse("15:04", strings.Trim(strArr[0], "\t"))
				if err != nil {
					return lineNum, fmt.Errorf("endtime parse fail")
				}
				e.Entry.Hours = time.Duration(e.Entry.EndTime.Sub(e.Entry.StartTime))

				db.SaveEntry(e)
			}
			// Start new entry
			e.Entry.Desc = ""
			e.Entry.ProjCode = strArr[1]
			e.Entry.StartTime, err = time.Parse("15:04", strings.Trim(strArr[0], "\t"))
			if err != nil {
				return lineNum, fmt.Errorf("startTime parse fail")
			}

		} else {
			// None indented are new dates.
			// If we hit a new date, the last potential entry should probably just be cleared.
			e = EntryRow{}
			//  There should be no spaces on this line, stripping will some some common errors.
			str = strings.Replace(str, " ", "", -1)
			e.Entry.Date, err = time.Parse("2006-01-02", str)
			if err != nil {
				return lineNum, fmt.Errorf("date parse fail")
			}
		}
		// fmt.Println(lineNum)
	}
}
