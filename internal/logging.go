package main

import (
	"io"
	"log"
	"os"
)

// Verbose is tied to the --verbose/-v flag; logger prints to /dev/null by default.
var Verbose bool

// logger prints messages only if we switch its output to stdout when verbose == true.
var logger = log.New(io.Discard, "", log.LstdFlags)

// EnableVerboseLogging is called once in the root command before we do any logging.
func EnableVerboseLogging() {
	if Verbose {
		logger.SetOutput(os.Stdout)
	}
}
