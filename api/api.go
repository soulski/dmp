package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/soulski/dmp/api/req"
	"github.com/soulski/dmp/api/res"
)

type HttpMethod string

const (
	GET    HttpMethod = "GET"
	POST   HttpMethod = "POST"
	PUT    HttpMethod = "PUT"
	DELETE HttpMethod = "DELETE"
)

var URLSchema = map[string]*Action{
	"GET:/namespace":                       action(listAllMember),
	"GET:/namespace/{namespace}":           action(listMember),
	"PUT:/namespace":                       action(serviceRegister),
	"DELETE:/namespace/{namespace}":        action(serviceUnregister),
	"PUT:/message/reqRes/{namespace}":      action(request),
	"PUT:/message/pubSub/{topic}":          action(publish),
	"PUT:/message/noti/{namespace}":        action(notificate),
	"PUT:/topic/{topicName}/subscriber":    action(subscribeTopic),
	"DELETE:/topic/{topicName}/subscriber": action(unsubscribeTopic),
}

type API interface {
	ServiceRegister(ns string, contactPoint string) (*res.Member, error)
	ServiceUnregister() bool
	ListMembers(ns string) *res.Members
	ListAllMembers() *res.Members
	Request(namespace string, msg []byte) ([]byte, error)
	Publish(topic string, msg []byte) ([]byte, error)
	Notificate(namespace string, msg []byte) ([]byte, error)
	SubscribeTopic(topicName string) bool
	UnsubscribeTopic(topicName string) bool
}

type Action struct {
	api    API
	method HttpMethod
	action func(api API, w http.ResponseWriter, req *http.Request)
}

func action(action func(api API, w http.ResponseWriter, req *http.Request)) *Action {
	return &Action{action: action}
}

func (a *Action) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	a.action(a.api, w, req)
}

type ApiServer struct {
	router *closableRouter
	api    API

	urlSchema map[string]Action

	running bool
	logger  *log.Logger
}

func CreateApiServer(api API, logger *log.Logger) *ApiServer {
	sMux := mux.NewRouter()

	for url, handler := range URLSchema {
		elems := strings.Split(url, ":")
		method, url := elems[0], elems[1]

		handler.api = api
		handler.method = HttpMethod(method)
		sMux.Handle(url, handler).Methods(method)
	}

	return &ApiServer{
		api:    api,
		logger: logger,
		router: &closableRouter{sMux, false},
	}
}

func (c *ApiServer) Start() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: c.router,
	}

	server.ListenAndServe()
}

func (c *ApiServer) Stop() error {
	c.router.Close()
	return nil
}

func RunAPI(api API, logger *log.Logger) (*ApiServer, chan bool, error) {
	started := make(chan bool)

	apiServ := CreateApiServer(api, logger)

	go func() {
		started <- true
		apiServ.Start()
	}()

	return apiServ, started, nil
}

type closableRouter struct {
	*mux.Router

	closed bool
}

func (r *closableRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.closed != true {
		r.Router.ServeHTTP(w, req)
	} else {
		fmt.Fprint(w, "connection closed")
	}
}

func (r *closableRouter) Close() {
	r.closed = true
}

func listAllMember(api API, w http.ResponseWriter, httpReq *http.Request) {
	result := api.ListAllMembers()
	if err := writeJSON(w, result); err != nil {
		http.Error(w, err.Error(), 403)
	}
}

func listMember(api API, w http.ResponseWriter, httpReq *http.Request) {
	result := api.ListMembers(mux.Vars(httpReq)["namespace"])
	if err := writeJSON(w, result); err != nil {
		http.Error(w, err.Error(), 403)
	}
}

func serviceRegister(api API, w http.ResponseWriter, httpReq *http.Request) {
	var service req.Service

	decoder := json.NewDecoder(httpReq.Body)
	if err := decoder.Decode(&service); err != nil {
		http.Error(w, err.Error(), 403)
		return
	}

	s, err := api.ServiceRegister(
		service.Namespace,
		service.ContactPoint,
	)

	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}

	writeJSON(w, s)
}

func serviceUnregister(api API, w http.ResponseWriter, httpReq *http.Request) {
	success := api.ServiceUnregister()
	writeJSON(w, &res.Result{Result: success})
}

func subscribeTopic(api API, w http.ResponseWriter, httpReq *http.Request) {
	topic := mux.Vars(httpReq)["topicName"]

	success := api.SubscribeTopic(topic)
	writeJSON(w, &res.Result{Result: success})
}

func unsubscribeTopic(api API, w http.ResponseWriter, httpReq *http.Request) {
	topic := mux.Vars(httpReq)["topicName"]

	success := api.UnsubscribeTopic(topic)
	writeJSON(w, &res.Result{Result: success})
}

func request(api API, w http.ResponseWriter, httpReq *http.Request) {
	params := mux.Vars(httpReq)

	ns, ok := params["namespace"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : namespace is required"))
	}

	b, err := ioutil.ReadAll(httpReq.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : " + err.Error()))
	}

	res, err := api.Request(ns, b)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : " + err.Error()))

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res); err != nil {
		fmt.Println("Error : ", err)
	}
}

func publish(api API, w http.ResponseWriter, httpReq *http.Request) {
	params := mux.Vars(httpReq)

	ns, ok := params["topic"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : namespace is required"))
	}

	b, err := ioutil.ReadAll(httpReq.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : " + err.Error()))
	}

	res, err := api.Publish(ns, b)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : " + err.Error()))

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res); err != nil {
		fmt.Println("Error : ", err)
	}
}

func notificate(api API, w http.ResponseWriter, httpReq *http.Request) {
	params := mux.Vars(httpReq)

	ns, ok := params["namespace"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : namespace is required"))
	}

	b, err := ioutil.ReadAll(httpReq.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : " + err.Error()))
	}

	res, err := api.Notificate(ns, b)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error : " + err.Error()))

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res); err != nil {
		fmt.Println("Error : ", err)
	}
}

func writeJSON(w http.ResponseWriter, obj interface{}) (err error) {
	if member, err := json.Marshal(obj); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(member)
	}

	return err
}
