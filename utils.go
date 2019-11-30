package main

func StringArrayContains(arr []string, v string) bool {
	for i := range arr {
		if arr[i] == v {
			return true
		}
	}
	return false
}
