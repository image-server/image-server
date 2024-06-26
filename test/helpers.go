package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"
)

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// Matches fails the test if act does not match exp.
func Matches(tb *testing.T, exp string, act string) {
	if !regexp.MustCompile(exp).MatchString(act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\nexpected: %#v\n\n\tto match: %#v\033[39m\n\n", filepath.Base(file), line, act, exp)
		tb.FailNow()
	}
}

// ExpectFile fails if the file does not exist
func ExpectFile(tb *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: Expected file to Exist: %s\033[39m\n\n", filepath.Base(file), line, err)
		tb.FailNow()
	}
}

func ReaderToString(input io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(input)
	return buf.String()
}
