package sCError

import "fmt"

const (
	COMPRESSOR_ERROR_SEVERITY_ERROR = "error"
	COMPRESSOR_ERROR_SEVERITY_INFO  = "info"
)

type CompressorError struct {
	Severity string
	Message  string
}

func (ce *CompressorError) Error() string {
	return fmt.Sprintf("(%s) %s", ce.Severity, ce.Message)
}
