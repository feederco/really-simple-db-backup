package pkg

import "log"

// VerboseMode is a global switch to turn verbose mode off or on
var VerboseMode bool

// Log is the default log to use
var Log *log.Logger

// ErrorLog is the default error log to use
var ErrorLog *log.Logger
