package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Export worklog function will take all rows from db and append to file.
// func ExportWorklog() (int, error) {
// 	// Read from db, format the read into a string, write newline.
// 	var entry EntryRow
// 	prevDate := time.Time{}
// 	startId := -1
// 	overrun := false
// 	f, err := os.OpenFile("export.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
// 	if err != nil {
// 		return -1, err
// 	}
// 	defer f.Close()
// 	w := bufio.NewWriter(f)
// 	for {
// 		if err != nil {
// 			return -1, err
// 		}
// 		if startId == -1 {
// 			startId = entry.entryId
// 		}
// 		if startId == entry.entryId && overrun {
// 			return 0, nil
// 		}
// 		overrun = true
// 		if prevDate != entry.entry.date {
// 			prevDate = entry.entry.date
// 			w.WriteString(fmt.Sprintf("%s\n", entry.entry.date.Format("2006-01-02")))
// 		}
// 		w.WriteString(fmt.Sprintf("\t%02d:%02d:%02d %s\n\t\t%s\n", int(entry.entry.hours.Hours()), int(entry.entry.hours.Minutes())%60, int(entry.entry.hours.Seconds())%60, entry.entry.projCode, entry.entry.desc))
// 	}
// }

// For the moment the import function will only look for a file
func ImportWorklog() (int, error) {
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
			e.entry.desc += str
			e.entry.desc += "\n"
		} else if strBytes[0] == '\t' && strBytes[1] != '\t' {
			// Time, proj code and tags on single indent lines.
			// Check that this isnt a line with nothing but a tab, happens
			if strBytes[1] == '\r' || strBytes[1] == '\n' {
				continue
			}
			// Finish old entry and submit.
			strArr := strings.Split(str, " ")

			if !e.entry.startTime.IsZero() {
				e.entry.endTime, err = time.Parse("15:04", strings.Trim(strArr[0], "\t"))
				if err != nil {
					return lineNum, fmt.Errorf("endtime parse fail")
				}
				e.entry.hours = time.Duration(e.entry.endTime.Sub(e.entry.startTime))

				db.SaveEntry(e)
			}
			// Start new entry
			e.entry.desc = ""
			e.entry.projCode = strArr[1]
			e.entry.startTime, err = time.Parse("15:04", strings.Trim(strArr[0], "\t"))
			if err != nil {
				return lineNum, fmt.Errorf("startTime parse fail")
			}

		} else {
			// None indented are new dates.
			// If we hit a new date, the last potential entry should probably just be cleared.
			e = EntryRow{}
			//  There should be no spaces on this line, stripping will some some common errors.
			str = strings.Replace(str, " ", "", -1)
			e.entry.date, err = time.Parse("2006-01-02", str)
			if err != nil {
				return lineNum, fmt.Errorf("date parse fail")
			}
		}
		// fmt.Println(lineNum)
	}
}
