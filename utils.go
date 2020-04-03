package main

import "os"

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
