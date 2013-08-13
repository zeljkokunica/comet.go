package l

import (
	"fmt"
	"log"	
)
type LogLevel struct {
	level int
	name string
}

var DEBUG = LogLevel{0, "DEBUG"}
var INFO = LogLevel{1, "INFO"}
var WARN = LogLevel{2, "WARN"}
var ERROR = LogLevel{3, "ERROR"}

var MIN_LOG_LEVEL = 0
	
func writeLog(level LogLevel, message string) {
	if level.level >= MIN_LOG_LEVEL { 
		log.Printf("[%s] %s", level.name, message)
	}
}

func writeLogF(level LogLevel, format string, v ...interface{}) {
	var message = fmt.Sprintf(format, v...)
	writeLog(level, message)
}
func D(message string) {
	writeLog(DEBUG, message)
}
func Df(format string, v ...interface{}) {
	writeLogF(DEBUG, format, v...)
}

func Wf(format string, v ...interface{}) {
	writeLogF(WARN, format, v...)
}

func W(message string) {
	writeLog(WARN, message)
}
func If(format string, v ...interface{}) {
	writeLogF(INFO, format, v...)
}
func I(message string) {
	writeLog(INFO, message)
}

func Ef(format string, v ...interface{}) {
	writeLogF(ERROR, format, v...)
}

func E(message string) {
	writeLog(ERROR, message)
}




