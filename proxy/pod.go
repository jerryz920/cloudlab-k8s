package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	logrus "github.com/sirupsen/logrus"
)

type Pod struct {
	Uuid string
	Tag  string
}

type PodRequest struct {
	Uuid string
	Tag  string `json:"tag,omitempty"`
	Addr string
}

func ParsePodRequest(r io.Reader) (*PodRequest, error) {
	d := json.NewDecoder(r)
	var p PodRequest
	if err := d.Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *MetadataProxy) postPodNew(w http.ResponseWriter, r *http.Request) {
	podReq, err := ParsePodRequest(r.Body)
	defer r.Body.Close()
	if err != nil {
		logrus.Error("Bad Pod Request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	addr := podReq.Addr
	uuid := podReq.Uuid
	c.pods[addr] = Pod{
		Uuid: uuid,
		Tag:  podReq.Tag,
	}

}

func (c *MetadataProxy) authPodNew(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addr := vars["addr"]
	if pod, ok := c.pods[addr]; ok {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s\n%s\n", pod.Uuid, pod.Tag)))
	} else {
		logrus.Error("fail to authenticate the network address")
		w.WriteHeader(http.StatusNotFound)
	}
}
