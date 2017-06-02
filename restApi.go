package main

import (
	"fmt"
	"net/http"
	"log"

	"github.com/gorilla/mux"
	"github.com/joyrexus/buckets"
	"encoding/json"
	"unicode/utf8"
)

type ErrorObj struct {
	Error bool `json:"error"`
	Msg string `json:"message"`
}

func (r ErrorObj) String() string {
	r.Error = true
	out, _ := json.Marshal(r)
	return string(out)
}

func Reverse(s string) string {
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}

type RestApi struct {
	port string
	db *buckets.Bucket
}

func (r RestApi) serve() {
	router := mux.NewRouter()

	router.HandleFunc("/routes", r.routesGet).Methods("GET")
	router.HandleFunc("/routes", r.routePost).Methods("POST")

	log.Printf("Rest API on port %s", r.port)
	http.ListenAndServe(":" + r.port, router)
}

func (r RestApi) routesGet(res http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		res.Header().Set("Access-Control-Allow-Origin", origin)
		res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}

	res.Header().Set("Content-Type", "application/json")

	var routes []RouteConfig

	items, _ := r.db.Items()
	for _, element := range items {
		route := RouteConfig{}
		route.Unmarshal(element.Value)

		routes = append(routes, route);
	}

	out, _ := json.Marshal(&routes)
	fmt.Fprint(res, string(out))
}

/*func (r RestApi) routeGet(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(req)

	value, err := r.db.Get([]byte(vars["url"]))
	if err != nil || value == nil {
		http.Error(res, ErrorObj{Msg: "Does not exist"}.String(), 404)
		return
	}
	log.Println(value)

	route := &RouteConfig{}
	err = route.Unmarshal(value)
	if err != nil {
		http.Error(res, ErrorObj{Msg: "Could not parse entry"}.String(), 503)
		return
	}

	fmt.Fprint(res, route.String())
}*/

func (r RestApi) routePost(res http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		res.Header().Set("Access-Control-Allow-Origin", origin)
		res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}

	res.Header().Set("Content-Type", "application/json")

	var route *RouteConfig
	err := json.NewDecoder(req.Body).Decode(&route)

	if err != nil {
		http.Error(res, ErrorObj{Msg: "Could not parse entry"}.String(), 503)
		return
	}
	req.Body.Close()

	value, err := route.Marshal()
	if err != nil {
		http.Error(res, ErrorObj{Msg: "Could not parse entry"}.String(), 503)
		return
	}

	err = r.db.Put([]byte(route.Url), value)
	if err != nil {
		http.Error(res, ErrorObj{Msg: "Could not insert entry"}.String(), 503)
		return
	}

	fmt.Fprint(res, route.String())
}