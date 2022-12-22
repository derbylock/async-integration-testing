package httputils

import (
	"encoding/json"
	"net/http"

	"github.com/derbylock/async-integration-testing/cmd/server/servererrors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

func WriteProtoJsonMessageOrError(w http.ResponseWriter, resp proto.Message, error error) {
	if error != nil {
		servererrors.SendInternalError(w, error)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	bytes, err := protojson.Marshal(resp)
	if err != nil {
		servererrors.SendInternalError(w, err)
		return
	}
	_, err = w.Write(bytes)
	if err != nil {
		servererrors.SendInternalError(w, err)
		return
	}
}

func WriteProtoArrayJsonMessageOrError[T proto.Message](w http.ResponseWriter, resps []T, error error) {
	if error != nil {
		servererrors.SendInternalError(w, error)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, err := w.Write([]byte("["))
	if err != nil {
		servererrors.SendInternalError(w, err)
		return
	}

	for i, resp := range resps {
		if i>0 {
			_, err = w.Write([]byte(","))
			if err != nil {
				servererrors.SendInternalError(w, err)
				return
			}
		}

		bytes, err := protojson.Marshal(resp)
		if err != nil {
			servererrors.SendInternalError(w, err)
			return
		}
		_, err = w.Write(bytes)
		if err != nil {
			servererrors.SendInternalError(w, err)
			return
		}
	}

	_, err = w.Write([]byte("]"))
	if err != nil {
		servererrors.SendInternalError(w, err)
		return
	}
}

func WriteJsonMessageOrError(w http.ResponseWriter, resp interface{}, error error) {
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
