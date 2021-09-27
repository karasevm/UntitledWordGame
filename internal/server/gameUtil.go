package server

import (
	"bufio"
	"embed"
	"fmt"
)

func readLines(path string, f embed.FS) ([]string, error) {
	file, err := f.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// log message msg with prefix tag, colored according to tagLevel
// levels from 0 - important, to 4 - verbose
func activityLog(tag string, tagLevel int, msg ...interface{}) {
	//TODO: add logging level according to prod/dev
	reset := "\033[0m"
	colors := []string{"\033[31m", "\033[33m", "\033[32m", "\033[34m", "\033[37m"}
	var msgString string
	for _, val := range msg {
		msgString += " " + fmt.Sprintf("%v", val)
	}
	fmt.Println(string(colors[tagLevel]), tag, string(reset), ": ", msgString)
}
