package main

import (
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
	"unsafe"
)

const (
	rs6Letters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	rs6LetterIdxBits = 6
	rs6LetterIdxMask = 1<<rs6LetterIdxBits - 1
	rs6LetterIdxMax  = 63 / rs6LetterIdxBits
)

// RandAlphaNumericString 指定した文字数のランダム英数字文字列を生成します
func RandAlphaNumericString(n int) string {
	b := make([]byte, n)
	cache, remain := rand.Int63(), rs6LetterIdxMax
	for i := n - 1; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), rs6LetterIdxMax
		}
		idx := int(cache & rs6LetterIdxMask)
		if idx < len(rs6Letters) {
			b[i] = rs6Letters[idx]
			i--
		}
		cache >>= rs6LetterIdxBits
		remain--
	}
	return *(*string)(unsafe.Pointer(&b))
}

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

type AccessLoggingFormatter struct {
	l *zap.Logger
}

// NewLogEntry implements LogFormatter interface
func (f *AccessLoggingFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &logEntry{
		AccessLoggingFormatter: f,
		req:                    r,
	}
}

type logEntry struct {
	*AccessLoggingFormatter
	req *http.Request
}

// Write implements LogEntry interface
func (e *logEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	e.l.Info(fmt.Sprintf("[%d] %s %s", status, e.req.Method, e.req.URL.Path),
		zap.Int("status", status),
		zap.String("method", e.req.Method),
		zap.String("path", e.req.URL.Path),
		zap.String("ip", e.req.RemoteAddr),
		zap.String("ua", e.req.UserAgent()),
		zap.Duration("latency", elapsed),
		zap.Int("respSize", bytes),
		zap.String("reqId", middleware.GetReqID(e.req.Context())),
	)
}

// Panic implements LogEntry interface
func (e *logEntry) Panic(v interface{}, stack []byte) {
	dump, _ := httputil.DumpRequest(e.req, false)
	e.l.Error("[Recovery from panic]",
		zap.Time("time", time.Now()),
		zap.Any("error", v),
		zap.String("request", string(dump)),
		zap.String("stack", string(stack)),
		zap.String("reqId", middleware.GetReqID(e.req.Context())),
	)
}
