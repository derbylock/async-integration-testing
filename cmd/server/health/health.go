package health

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/derbylock/async-integration-testing/cmd/server/servererrors"
	"github.com/julienschmidt/httprouter"
)

type healthResponse struct {
	Status   string `json:"status"`
	Revision string `json:"revision"`
}

func InitAPIRoutes(pathPrefix string, router *httprouter.Router) {
	router.GET(pathPrefix+"/health", GetHealthRoute)
}

func GetHealthRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	revision := os.Getenv("REVISION")
	resp := healthResponse{
		Status:   "Healthy",
		Revision: revision}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		servererrors.SendInternalError(w, err)
	}
}
