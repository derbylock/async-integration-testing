package requestlogger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type RequestLoggingKey struct {
	Name string
}

type LogRecord struct {
	Type       string `json:"type"`
	RequestId  string `json:"rid"`
	RemoteAddr string `json:"raddr"`
	Time       string `json:"time"`
	Url        string `json:"url"`
	Path       string `json:"path"`
	Host       string `json:"host"`
	Method     string `json:"method"`
	Proto      string `json:"proto"`
	Status     int    `json:"status"`
	Written    int64  `json:"written"`
	Referer    string `json:"ref"`
	UserAgent  string `json:"uag"`
}

const RequestIdHeaderName = "X-ASIT-REQUESTID"

var requestCounter uint64
var RequestIdLoggingKey = RequestLoggingKey{Name: "rid"}
var baseRequestIdPrefix = uuid.New().String()

func Logger(out io.Writer, h http.Handler) http.Handler {
	logger := log.New(out, "", 0)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get(RequestIdHeaderName)
		if requestId == "" {
			atomicId := atomic.AddUint64(&requestCounter, 1)
			requestId = fmt.Sprintf("%s-%d", baseRequestIdPrefix, atomicId)
		}

		o := &responseObserver{ResponseWriter: w, requestId: requestId}
		h.ServeHTTP(o, r)
		addr := r.RemoteAddr
		if i := strings.LastIndex(addr, ":"); i != -1 {
			addr = addr[:i]
		}

		record := LogRecord{
			Type:       "stat",
			RequestId:  requestId,
			RemoteAddr: addr,
			Time:       time.Now().Format(time.RFC3339),
			Url:        trunc(r.URL.RequestURI()),
			Path:       trunc(r.URL.Path),
			Host:       trunc(r.Host),
			Method:     trunc(r.Method),
			Proto:      trunc(r.Proto),
			Status:     o.status,
			Written:    o.written,
			Referer:    trunc(r.Referer()),
			UserAgent:  trunc(r.UserAgent()),
		}

		jsonBytes, err := json.Marshal(record)
		if err != nil {
			// TODO: add monitoring for failed logs writing
			return
		}
		logger.Println(string(jsonBytes))
	})
}

type responseObserver struct {
	http.ResponseWriter
	requestId   string
	status      int
	written     int64
	wroteHeader bool
}

func (o *responseObserver) Write(p []byte) (n int, err error) {
	o.ResponseWriter.Header().Set(RequestIdHeaderName, o.requestId)
	if !o.wroteHeader {
		o.WriteHeader(http.StatusOK)
	}
	n, err = o.ResponseWriter.Write(p)
	o.written += int64(n)
	return
}

func (o *responseObserver) WriteHeader(code int) {
	o.ResponseWriter.Header().Set(RequestIdHeaderName, o.requestId)
	o.ResponseWriter.WriteHeader(code)
	if o.wroteHeader {
		return
	}
	o.wroteHeader = true
	o.status = code
}

func trunc(str string) string {
	return truncateStart(str, 256, "...")
}

func truncateStart(str string, length int, omission string) string {
	r := []rune(str)
	sLen := len(r)
	if length >= sLen {
		return str
	}
	return string(omission + string(r[len(r)-length+utf8.RuneCountInString(omission):]))
}
