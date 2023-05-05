package main

import (
	"os"
	"strings"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func safeConvertString(b []byte) string {
	bld := strings.Builder{}
	bld.Grow(len(b))
	for _, c := range string(b) {
		if c == '\uFFFD' {
			bld.WriteRune('.')
		} else {
			bld.WriteRune(c)
		}
	}
	return bld.String()
}
