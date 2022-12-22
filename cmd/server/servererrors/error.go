package servererrors

import (
	"encoding/json"
	"net/http"
)

const RequestIdHeaderName = "X-ASIT-REQUESTID"
const ErrorHeaderName = "X-ASIT-ERROR"

type errorJSON struct {
	ErrorMessage string `json:"message"`
}

func RenderError(w http.ResponseWriter, message string, statusCode int) {
	responseJSON := errorJSON{ErrorMessage: message}

	response, err := json.Marshal(responseJSON)
	if err != nil {
		SendInternalError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Connection", "close")
	w.WriteHeader(statusCode)
	w.Write(response)
}

func SendInternalError(w http.ResponseWriter, err error) {
	w.Header().Set(ErrorHeaderName, err.Error())
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusServiceUnavailable)
}

func SendInvalidJSON(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
}

func SendBadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

func SendEntityNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func SendConflictError(w http.ResponseWriter, err error) {
	if err != nil {
		w.Header().Set(ErrorHeaderName, err.Error())
	}
	w.WriteHeader(http.StatusConflict)
}
