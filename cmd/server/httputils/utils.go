package httputils

import (
	"encoding/json"
	"net/http"

	"github.com/derbylock/async-integration-testing/cmd/server/servererrors"
)

func WriteJson(w http.ResponseWriter, resp *interface{}) {
	if resp == nil {
		servererrors.SendEntityNotFound(w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		servererrors.SendInternalError(w, err)
	}
}

func WriteJsonProtoMessageOrError(w http.ResponseWriter, resp interface{}, error error) {
	if error != nil {
		servererrors.SendInternalError(w, error)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		servererrors.SendInternalError(w, err)
	}
}

func WriteNoContentOrError(w http.ResponseWriter, error error) {
	if error != nil {
		servererrors.SendInternalError(w, error)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
