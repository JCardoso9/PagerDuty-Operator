package pd_errors

import (
	"errors"
	"fmt"
	"runtime"
)

func New(err string) error {
	return errors.New(err)
}

// getCallerInfo returns file name, line number, function name
// It uses caller to get info about which function caused to call error handling mechanism
// for more: https://golang.org/pkg/runtime/#Caller
func getCallerInfo() (string, int, string) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "?", 0, "?"
	}

	fn := runtime.FuncForPC(pc)
	return file, line, fn.Name()
}

// Wrap allows you to wrap the error message with
// file,line,caller informations which can be useful for reporting.
// Nested errors will be wrapped as well.
func Wrap(err error, message string) error {
	f, l, fn := getCallerInfo()
	if err != nil {
		return fmt.Errorf("File: %v \nLine: %v \nCaller: %v \nMessage: %s \n%w\n\n", f, l, fn, message, err)
	}
	return nil
}
