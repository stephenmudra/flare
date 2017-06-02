package main

import (
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
)

type RouteConfig struct {
	Url string `json:"url"`
	Type string `json:"type"` // "forwarding", "static"
	Active bool `json:"active"`

	Nameservers []string `json:"nameservers"` // [ "8.8.8.8", "8.8.4.4" ]

	// "type": "static"
	Addrs  []string   `json:"addrs"`
	Cnames []string   `json:"cnames"`
	Txts   [][]string `json:"txts"`
}

func (r RouteConfig) Marshal() ([]byte, error) {
	return bson.Marshal(&r)
}

func (r *RouteConfig) Unmarshal(in []byte) (error) {
	return bson.Unmarshal(in, r)
}

func (r RouteConfig) String() string {
	out, _ := json.Marshal(&r)
	return string(out)
}
