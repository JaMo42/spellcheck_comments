// Package common contains types and utilities needed by all other packages.
package common

import (
	"fmt"
	"log"
)

var (
	InvocationName string
)

// Fatal aborts the program with an errors message that is prefixed with the
// invocation name of the program.
func Fatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Fatalf("%s: %s\n", InvocationName, msg)
}
