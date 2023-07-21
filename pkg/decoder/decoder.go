package decoder

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Task struct {
	projCode  string
	tags      []string
	data      string
	startTime time.Time
	endTime   time.Time
}

type Day struct {
	tasks []Task
	date  time.Time
}

type Overview struct {
	allEntries []Day
}

type Decode interface {
	DecodeDay(string) []Task
	DecodeTask() Day
	decodeWorklog(string) []Task
	GetDay()
}

type Util interface {
	PrintDates()
	PrintProjects()
	PrintTime()
}

func (d Day) GetDay() Day {
	return d
}

func (d Day) decodeDay(lines []string) (nextDay Day, day Day) {
	// the returned next task is the entered current task.
	layout := "2006-01-02"
	timeout := "15:04"
	var nextTask = Task{}
	newTime := 0

	for _, line := range lines {
		split := strings.Fields(line)

		if len(split) != 0 {
			//fmt.Println(line)
			if strings.HasPrefix(line, "\t") {
				//fmt.Println("pref")
				dateTime := split[0]
				hm, err := time.Parse(timeout, dateTime)
				//fmt.Println(err)
				if err != nil {
					//fmt.Println("data")
					d.tasks[newTime].data = line
				} else {
					//fmt.Println(err)
					if len(d.tasks) == 0 {
						d.tasks = append(d.tasks, nextTask)

						// empty means new day
						d.tasks[newTime].startTime = hm
						d.tasks[newTime].projCode = split[1]
						d.tasks[newTime].tags = split[2:]

					} else {
						d.tasks[newTime].endTime = hm
						// Needs a check to make sure buffer doesn't overflow since we might not have a another line.
						// going to try appending.
						nextTask = Task{startTime: hm, projCode: split[1], tags: split[2:]}
						d.tasks = append(d.tasks, nextTask)

						newTime++
					}

				}
			} else if check, err := time.Parse(layout, split[0]); err == nil {
				//fmt.Println("new date")

				if d.date.IsZero() {
					d.date = check
				} else {
					nextDay.date = check
					//fmt.Println(check)

				}
			}
		}
	}

	// if endTime isZero == true, not a task but an leave time.
	day = d
	return
}

func (o Overview) PrintDates() {
	for _, val := range o.allEntries {
		fmt.Println(val.date)
		//fmt.Println("Done decode")

	}
}

func DecodeWorklog(filename string) (days []Day) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	firstDate := true

	scanner := bufio.NewScanner(file)

	days = []Day{}
	day := Day{}
	next := Day{}
	layout := "2006-01-02"
	var entry []string
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 0 {

			if strings.HasPrefix(line, "\t") {
				entry = append(entry, line)

			} else if taskTime, err := time.Parse(layout, parts[0]); err == nil {
				if firstDate {
					entry = append(entry, line)
					firstDate = false
				} else {
					entry = append(entry, line)
					next, day = day.decodeDay(entry)
					next.date = taskTime
					days = append(days, day)
					day = next
					entry = []string{}
					//firstDate = true
				}
			} else {
				entry = append(entry, line)
			}
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}
	//fmt.Println(days)
	return
}
