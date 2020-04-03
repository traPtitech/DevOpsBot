package main

import (
	"math/rand"
	"os"
	"sync"
	"time"
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

var randSrcPool = sync.Pool{
	New: func() interface{} {
		return rand.NewSource(time.Now().UnixNano())
	},
}

const (
	rs6Letters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	rs6LetterIdxBits = 6
	rs6LetterIdxMask = 1<<rs6LetterIdxBits - 1
	rs6LetterIdxMax  = 63 / rs6LetterIdxBits
)

// RandAlphabetAndNumberString 指定した文字数のランダム英数字文字列を生成します
func RandAlphabetAndNumberString(n int) string {
	b := make([]byte, n)
	randSrc := randSrcPool.Get().(rand.Source)
	cache, remain := randSrc.Int63(), rs6LetterIdxMax
	for i := n - 1; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), rs6LetterIdxMax
		}
		idx := int(cache & rs6LetterIdxMask)
		if idx < len(rs6Letters) {
			b[i] = rs6Letters[idx]
			i--
		}
		cache >>= rs6LetterIdxBits
		remain--
	}
	randSrcPool.Put(randSrc)
	return string(b)
}
