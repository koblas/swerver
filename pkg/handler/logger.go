package handler

import "log"

type Logger interface {
	Debug(string, ...interface{})
}

type fullLogger struct{}
type stubLogger struct{}

// NewLogger is a hack to enable/disable logging quickly without putting
// the logic throughout the code
func NewLogger(debug bool) Logger {
	if debug {
		return fullLogger{}
	}

	return stubLogger{}
}

//
func (stubLogger) Debug(string, ...interface{}) {
}

func (fullLogger) Debug(msg string, args ...interface{}) {
	data := []interface{}{msg}
	data = append(data, args...)
	log.Println(data...)
}
