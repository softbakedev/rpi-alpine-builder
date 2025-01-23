package internal

import (
	"io"
	"log"
	"os"
)

// logger prints messages only if we switch its output to stdout when verbose == true.
var logger = log.New(io.Discard, "", log.LstdFlags)

// EnableVerboseLogging is called once in the root command before we do any logging.
func EnableVerboseLogging(verbose bool) {
	if verbose {
		logger.SetOutput(os.Stdout)
	}
}
