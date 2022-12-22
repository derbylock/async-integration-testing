package asit_api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	srv "github.com/derbylock/async-integration-testing/cmd/server/httputils"
	srvErrors "github.com/derbylock/async-integration-testing/cmd/server/servererrors"
	"github.com/derbylock/async-integration-testing/internal/db"
	"github.com/derbylock/async-integration-testing/pkg/asit"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const MAX_KEY_SIZE = 1024

type ClientsAPIController struct {
	clientsRepository db.ClientsRepository
}

func NewClientsAPIController(clientsRepository db.ClientsRepository) *ClientsAPIController {
	return &ClientsAPIController{clientsRepository: clientsRepository}
}

func (c *ClientsAPIController) InitRoutes(pathPrefix string, router *httprouter.Router) {
	router.GET(pathPrefix+"/clients", c.GetAllClientsHandler)
	router.POST(pathPrefix+"/clients", c.AddClientHandler)
	router.GET(pathPrefix+"/clients/:clientId", c.GetClientHandler)
	router.PUT(pathPrefix+"/clients/:clientId", c.UpdateClientHandler)
	router.DELETE(pathPrefix+"/clients/:clientId", c.DeleteClientHandler)

	router.GET(pathPrefix+"/clients/:clientId/keys", c.GetAllClientKeysHandler)
	router.POST(pathPrefix+"/client_keys/:key", c.AddClientKeyHandler)
	router.GET(pathPrefix+"/client_keys/:key", c.GetClientByKeyHandler)
	router.DELETE(pathPrefix+"/client_keys/:key", c.DeleteClientKeyHandler)
}

func (c *ClientsAPIController) GetAllClientsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	allClients, err := c.clientsRepository.GetAllClients(r.Context())
	srv.WriteJsonProtoMessageOrError(w, &asit.ClientList{Clients: allClients}, err)
}

func (c *ClientsAPIController) AddClientHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var client asit.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		srvErrors.SendInvalidJSON(w, err)
		return
	}

	newId, err := uuid.NewUUID()
	if err != nil {
		srvErrors.SendInternalError(w, err)
		return
	}
	client.Id = newId.String()
	client.LastUpdated = timestamppb.New(time.Now())
	err = c.clientsRepository.SetClient(r.Context(), &client)
	srv.WriteJsonProtoMessageOrError(w, &client, err)
}

func (c *ClientsAPIController) GetClientHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	clientId := params.ByName("clientId")
	client, err := c.clientsRepository.GetClientById(r.Context(), clientId)
	if client == nil {
		srvErrors.SendEntityNotFound(w)
		return
	}
	srv.WriteJsonProtoMessageOrError(w, client, err)
}

func (c *ClientsAPIController) DeleteClientHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	clientId := params.ByName("clientId")
	client, err := c.clientsRepository.GetClientById(r.Context(), clientId)
	if err != nil {
		srvErrors.SendInternalError(w, err)
		return
	}
	if client == nil {
		srvErrors.SendEntityNotFound(w)
		return
	}
	err = c.clientsRepository.RemoveClient(r.Context(), clientId)
	srv.WriteNoContentOrError(w, err)
}

func (c *ClientsAPIController) UpdateClientHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	clientId := params.ByName("clientId")
	client, err := c.clientsRepository.GetClientById(r.Context(), clientId)
	if client == nil {
		srvErrors.SendEntityNotFound(w)
		return
	}

	var newClient asit.Client
	if err := json.NewDecoder(r.Body).Decode(&newClient); err != nil {
		srvErrors.SendInvalidJSON(w, err)
		return
	}

	client.ClientProperties = newClient.ClientProperties
	c.clientsRepository.SetClient(r.Context(), client)
	srv.WriteJsonProtoMessageOrError(w, &client, err)
}

func (c *ClientsAPIController) GetAllClientKeysHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	clientId := params.ByName("clientId")
	allKeys, err := c.clientsRepository.GetClientKeys(r.Context(), clientId)
	if allKeys == nil {
		allKeys = &asit.ClientKeys{}
	}
	srv.WriteJsonProtoMessageOrError(w, allKeys, err)
}

func (c *ClientsAPIController) GetClientByKeyHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	key := params.ByName("key")
	client, err := c.clientsRepository.GetClientByKey(r.Context(), key)
	if client == nil {
		srvErrors.SendEntityNotFound(w)
		return
	}
	srv.WriteJsonProtoMessageOrError(w, client, err)
}

func (c *ClientsAPIController) AddClientKeyHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	key := params.ByName("key")
	if key == "" || len(key) > MAX_KEY_SIZE {
		srvErrors.SendBadRequest(w)
		return
	}
	var client asit.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		srvErrors.SendInvalidJSON(w, err)
		return
	}
	err := c.clientsRepository.AddClientKey(r.Context(), client.Id, key)
	if _, isConflictError := errors.Unwrap(err).(*db.NonUniqueClientKeyError); isConflictError {
		srvErrors.SendConflictError(w, err)
		return
	}
	srv.WriteNoContentOrError(w, err)
}

func (c *ClientsAPIController) DeleteClientKeyHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	key := params.ByName("key")
	err := c.clientsRepository.RemoveClientKey(r.Context(), key)
	srv.WriteNoContentOrError(w, err)
}
