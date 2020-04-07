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
	logrus.Infof("Pod post req: %v", *podReq)

	addr := podReq.Addr
	uuid := podReq.Uuid
	c.pods[addr] = Pod{
		Uuid: uuid,
		Tag:  podReq.Tag,
	}

}

func (c *MetadataProxy) authPodInner(addr string) *Pod {
	if pod, ok := c.pods[addr]; ok {
		logrus.Infof("Pod auth req for %s: %v", addr, pod)
		return &pod
	} else {
		return nil
	}

}

func (c *MetadataProxy) authPodNew(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	saddr := vars["addr"]

	if pod := c.authPodInner(saddr); pod != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s\n%s\n", pod.Uuid, pod.Tag)))
		return
	}
	addrPort := saddr
	addr, _, _, status := ParseIP2(addrPort)
	if status != http.StatusOK || addr == "" {
		logrus.Errorf("must provide a valid ip address: %v", addrPort)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if pod := c.authPodInner(addr); pod != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s\n%s\n", pod.Uuid, pod.Tag)))
	} else {
		logrus.Errorf("fail to authenticate the network address: [%v]", saddr)
		w.WriteHeader(http.StatusNotFound)
	}
}
