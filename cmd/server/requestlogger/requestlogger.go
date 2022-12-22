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

	"github.com/derbylock/async-integration-testing/cmd/server/servererrors"
	"github.com/derbylock/async-integration-testing/cmd/server/stringutils"
	"github.com/google/uuid"
)

type RequestLoggingKey struct {
	Name string
}

type LogRecord struct {
	Type       string `json:"type,omitempty"`
	RequestId  string `json:"rid,omitempty"`
	RemoteAddr string `json:"raddr,omitempty"`
	Time       string `json:"time,omitempty"`
	Url        string `json:"url,omitempty"`
	Path       string `json:"path,omitempty"`
	Host       string `json:"host,omitempty"`
	Method     string `json:"method,omitempty"`
	Proto      string `json:"proto,omitempty"`
	Status     int    `json:"status,omitempty"`
	Written    int64  `json:"written,omitempty"`
	Referer    string `json:"ref,omitempty"`
	UserAgent  string `json:"uag,omitempty"`
	Error      string `json:"err,omitempty"`
}

var requestCounter uint64
var RequestIdLoggingKey = RequestLoggingKey{Name: "rid"}
var baseRequestIdPrefix = uuid.New().String()

func Logger(out io.Writer, h http.Handler) http.Handler {
	logger := log.New(out, "", 0)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get(servererrors.RequestIdHeaderName)
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
			Url:        trunc(r.URL.RequestURI(), 256),
			Path:       trunc(r.URL.Path, 256),
			Host:       trunc(r.Host, 128),
			Method:     trunc(r.Method, 16),
			Proto:      trunc(r.Proto, 16),
			Status:     o.status,
			Written:    o.written,
			Referer:    trunc(r.Referer(), 256),
			UserAgent:  trunc(r.UserAgent(), 256),
			Error:      o.Header().Get(servererrors.ErrorHeaderName),
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
	o.ResponseWriter.Header().Set(servererrors.RequestIdHeaderName, o.requestId)
	if !o.wroteHeader {
		o.WriteHeader(http.StatusOK)
	}
	n, err = o.ResponseWriter.Write(p)
	o.written += int64(n)
	return
}

func (o *responseObserver) WriteHeader(code int) {
	o.ResponseWriter.Header().Set(servererrors.RequestIdHeaderName, o.requestId)
	o.ResponseWriter.WriteHeader(code)
	if o.wroteHeader {
		return
	}
	o.wroteHeader = true
	o.status = code
}

func trunc(str string, maxlen int) string {
	return stringutils.TruncateStart(str, maxlen, "...")
}
