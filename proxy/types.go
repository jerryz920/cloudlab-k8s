package main

import (
	"bytes"
	"encoding/json"
	"net"
	"time"

	logrus "github.com/sirupsen/logrus"
)

type Principal struct {
	Name    string
	ImageID string
	IP      string
	Config  string
	PortMin int
	PortMax int
}

type MetadataRequest struct {
	Principal   string   `json:"principal,omitempty"`
	ParentBear  string   `json:"bearerRef,omitempty"`
	OtherValues []string `json:"methodParams"`
	Auth        string   `json:"auth,omitempty"`

	/// fields not to be marshaled but used internally
	/// FIXME: duplicated stuff, can remove in future to use cache only
	ip    net.IP
	lport int
	rport int

	// Used to carry some intermediate information
	targetCidr      *net.IPNet
	targetIp        net.IP
	targetLport     int
	targetRport     int
	targetType      string
	targetAddrIndex int

	// original request
	url      string
	method   string
	instance *CachedInstance
	cache    Cache /// The cache to use for this request
}

func EncodingMetadataRequest(mr *MetadataRequest) (*bytes.Buffer, error) {
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(mr); err != nil {
		logrus.Debug("error encoding the principal ", err)
		return nil, err
	}
	return &buf, nil
}

type PrincipalResponse struct {
	Message string
}

type BearerMetadataRequest struct {
	Principal   string   `json:"principal"`
	BearerRef   string   `json:"bearerRef"`
	OtherValues []string `json:"otherValues"`
}

/* Only VM has Cidr, others don't */
type InstanceCred struct {
	Pid  string
	PPid string
	Cidr *net.IPNet
	Type string
}

type CachedInstance struct {
	Ip    net.IP
	Lport int
	Rport int
	ID    *InstanceCred
}

type CachedPod struct {
	Ip          net.IP
	Expire      time.Time
	PodInstance Pod
}
