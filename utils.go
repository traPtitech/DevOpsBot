package main

import (
	"os"
	"strings"
)

func StringArrayContains(arr []string, v string) bool {
	for i := range arr {
		if arr[i] == v {
			return true
		}
	}
	return false
}

func stringOrEmpty(s ...string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func getEnvOrDefault(env, def string) string {
	s := os.Getenv(env)
	if len(s) > 0 {
		return s
	}
	return def
}

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
