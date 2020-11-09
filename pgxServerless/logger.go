package pgxServerless

import (
	"encoding/json"
	"os"
)

const (
	errorLvl  = "ERROR"
	normalLvl = "NORMAL"
)

type logStructure struct {
	Level   string      `json:"level"`
	Message interface{} `json:"message"`
}

type Logger interface {
	Info(message interface{})
	Failure(err error)
}

type logger struct {
	level   string
	pid     int
	debug   bool
	message interface{}
	enc     *json.Encoder
}

func newLogger(debug bool) logger {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return logger{
		enc:   enc,
		debug: debug,
	}
}

func (l logger) Info(message interface{}) {
	fullMsg := logStructure{
		Level:   normalLvl,
		Message: message,
	}
	if l.debug {
		l.enc.Encode(fullMsg)
	}
}

func (l logger) Failure(err error) {
	msg := logStructure{
		Level:   errorLvl,
		Message: err.Error(),
	}
	if l.debug {
		l.enc.Encode(msg)
	}
}
