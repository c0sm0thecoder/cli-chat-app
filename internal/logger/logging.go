package logger

import (
	"log"
	"os"
)

func init() {
	// Set up logging format with timestamp and file location
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Optionally, write logs to a file
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	// Set output to both file and stdout
	log.SetOutput(logFile)
	log.SetOutput(os.Stdout)
}
