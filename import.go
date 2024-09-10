package main

import (
	"bufio"
	"os"
)

var data = make([]byte, 400)

func readFile() error {
	file, err := os.Open("worklog.txt")
	if err != nil {
		return err
	}
	rd := bufio.NewReader(file)
	rd.ReadString('\n')
	rd.R

	return nil
}
