package transcribe

import (
	"io"
)

// Result TODO
type Result struct {
	Text       string  `json:"text"`
	Confidence float32 `json:"confidence"`
	Final      bool    `json:"final"`
}

// Service TODO
type Service interface {
	CreateStream() (Stream, error)
}

// Stream TODO
type Stream interface {
	io.Writer
	io.Closer
	Results() <-chan Result
}
