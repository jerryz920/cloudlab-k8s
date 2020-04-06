package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	riak "github.com/basho/riak-go-client"
	"github.com/sirupsen/logrus"
)

const (
	NETMAP_PID  = "Pid_s"
	NETMAP_ID   = "Id_s"
	NETMAP_CIDR = "Cidr_s"
	NETMAP_TYPE = "Type_s"
)

type NetMapObj struct {
	Pid  string `json:"Id_s"`
	PPid string `json:"Pid_s"`
	Cidr string `json:"Cidr_s"`
	Type string `json:"Type_s"`
}

func InstanceCredFromBytes(data []byte) *InstanceCred {

	buf := bytes.NewBuffer(data)
	decoder := json.NewDecoder(buf)
	var obj NetMapObj
	if err := decoder.Decode(&obj); err != nil {
		logrus.Error("decoding Json: ", err)
		return nil
	}

	inst := InstanceCred{
		Pid:  obj.Pid,
		PPid: obj.PPid,
		Type: obj.Type,
	}

	if obj.Cidr == "" {
		inst.Cidr = nil
	} else {
		_, cidr, err := net.ParseCIDR(obj.Cidr)
		if err != nil {
			logrus.Errorf("parsing CIDR of %s: %v", obj.Cidr, err)
			return nil
		}
		inst.Cidr = cidr
	}

	return &inst
}

func (i *InstanceCred) Bytes() []byte {
	obj := NetMapObj{
		Pid:  i.Pid,
		PPid: i.PPid,
		Type: i.Type,
	}
	if i.Cidr == nil {
		obj.Cidr = ""
	} else {
		obj.Cidr = i.Cidr.String()
	}

	buf := bytes.NewBuffer(nil)
	decoder := json.NewEncoder(buf)
	if err := decoder.Encode(&obj); err != nil {
		logrus.Error("encoding Json")
		return nil
	}
	return buf.Bytes()
}

type RiakConn interface {
	Connect(addrs []string) error
	PutNetIDMap(ip net.IP, lport int, rport int, uuid *InstanceCred) error
	GetNetIDMap(ip net.IP, lport int, rport int) (*InstanceCred, error)
	DelNetIDMap(ip net.IP, lport int, rport int) error
	GetAllNetID(ip net.IP) ([]CachedInstance, error)
	PutPodMap(ip net.IP, pod *CachedPod) error
	// GetPod will delete expired pod.
	GetPodMap(ip net.IP) (*CachedPod, error)
	///
	SearchIDNet(uuid string) ([]CachedInstance, error)
	Shutdown() error
}

type riakConn struct {
	Addrs  []string
	Client *riak.Client
	/// TODO: adding settings for replications
}

func NewRiakConn() RiakConn {
	//riak.EnableDebugLogging = true
	return &riakConn{Addrs: []string{}, Client: nil}
}

func NetMapKey(lport, rport int) string {
	return fmt.Sprintf("%d:%d", lport, rport)
}

func (c *riakConn) checkIndex(indexName string) error {
	cmd, err := riak.NewFetchIndexCommandBuilder().
		WithIndexName(indexName).Build()
	if err != nil {
		logrus.Debug("building the fetch index cmd ", err)
		return err
	}

	//// If no error, check if the index really exists
	if err = c.Client.Execute(cmd); err == nil {
		logrus.Debug("valiadting existing index")
		result := cmd.(*riak.FetchIndexCommand).Response
		if result != nil && len(result) > 0 && result[0].Name == indexName {
			logrus.Debugf("Index %s exists", indexName)
			return nil
		}
	}

	cmd, err = riak.NewStoreIndexCommandBuilder().
		WithIndexName(indexName).
		WithTimeout(time.Second * 10).
		Build()
	if err != nil {
		logrus.Debug("building the create index cmd ", err)
		return err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("creating index: ", err)
		return err
	}
	logrus.Infof("Index %s created", indexName)

	cmd, err = riak.NewStoreBucketTypePropsCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithSearchIndex(RIAK_INDEX_NAME).
		WithSearch(true).Build()

	if err != nil {
		logrus.Debug("building the store bucket type property cmd ", err)
		return err
	}
	logrus.Info("associating bucket type index")
	return c.Client.Execute(cmd)
}

func (c *riakConn) Connect(addrs []string) error {
	if c.Client != nil {
		if err := c.Client.Stop(); err != nil {
			logrus.Errorf("can not stop previous riak conn to ", c.Addrs)
			return err
		}
	}
	options := riak.NewClientOptions{
		RemoteAddresses: addrs,
	}
	client, err := riak.NewClient(&options)
	if err != nil {
		logrus.Errorf("can not connect to given address %v", addrs)
		return err
	}
	c.Addrs = addrs
	c.Client = client
	logrus.Info("Checking index on the bucket type")

	/// check the indexes on the bucket type

	return c.checkIndex(RIAK_INDEX_NAME)
}

func (c *riakConn) DelNetAllocation(ip net.IP) error {
	cmd, err := riak.NewDeleteValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(CIDR_BUCKET).
		WithKey(ip.String()).
		Build()
	if err != nil {
		return err
	}
	err = c.Client.Execute(cmd)
	return err
}

func (c *riakConn) PutNetIDMap(ip net.IP, lport int, rport int, uuid *InstanceCred) error {
	obj := &riak.Object{
		ContentType:     "application/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		BucketType:      RIAK_BUCKET_TYPE,
		Bucket:          ip.String(),
		Key:             NetMapKey(lport, rport),
		Value:           uuid.Bytes(),
	}
	cmd, err := riak.NewStoreValueCommandBuilder().
		WithContent(obj).Build()
	if err != nil {
		logrus.Debug("error in building PutNetIDMap cmd")
		return err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error in executing PutNetIDMap")
		return err
	}
	return nil
}

func (c *riakConn) GetNetIDMap(ip net.IP, lport int, rport int) (*InstanceCred, error) {

	cmd, err := riak.NewFetchValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(ip.String()).
		WithKey(NetMapKey(lport, rport)).
		Build()
	if err != nil {
		logrus.Debug("error in building fetch net id map command")
		return nil, err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error executing fetch net id map command")
		return nil, err
	}

	if actual, ok := cmd.(*riak.FetchValueCommand); ok {
		if len(actual.Response.Values) == 0 {
			return nil, nil
		} else if len(actual.Response.Values) > 1 {
			logrus.Error("there is more than one UUID: BUG")
			for _, o := range actual.Response.Values {
				logrus.Info("%v", string(o.Value))
			}
		}
		return InstanceCredFromBytes(actual.Response.Values[0].Value), nil
	}
	logrus.Debug("error in reading response of get net id map")
	return nil, errors.New("Unknown command")
}

func (c *riakConn) DelNetIDMap(ip net.IP, lport, rport int) error {
	cmd, err := riak.NewDeleteValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(ip.String()).
		WithKey(NetMapKey(lport, rport)).
		Build()
	if err != nil {
		return err
	}
	err = c.Client.Execute(cmd)
	return err
}

func (c *riakConn) genCachedInstance(d *riak.SearchDoc) *CachedInstance {

	var lport, rport int
	if n, err := fmt.Sscanf(d.Key, "%d:%d", &lport, &rport); err != nil || n != 2 {
		logrus.Error("Error parsing the key, bug: ", d.Key)
		return nil
	}

	pids, ok := d.Fields[NETMAP_PID]
	if !ok || pids == nil || len(pids) == 0 {
		logrus.Error("missing ParentID in index: bucket=%s,key=%s, [%v]",
			d.Bucket, d.Key, d.Fields)
		return nil
	}
	pid := pids[0]

	ids, ok := d.Fields[NETMAP_ID]
	if !ok || ids == nil || len(ids) == 0 {
		logrus.Error("missing ID in index: bucket=%s,key=%s, [%v]",
			d.Bucket, d.Key, d.Fields)
		return nil
	}
	id := ids[0]

	cred := InstanceCred{
		Pid:  id,
		PPid: pid,
	}

	types, ok := d.Fields[NETMAP_TYPE]
	if !ok || types == nil || len(types) == 0 {
		logrus.Error("missing type in index: bucket=%s,key=%s, [%v]",
			d.Bucket, d.Key, d.Fields)
		cred.Type = NORMAL_INSTANCE_TYPE
	} else {
		cred.Type = types[0]
	}

	cidrs, ok := d.Fields[NETMAP_CIDR]
	if !ok || cidrs == nil || len(cidrs) == 0 || cidrs[0] == "" {
		cred.Cidr = nil
	} else {
		_, cidr, err := net.ParseCIDR(cidrs[0])
		if err != nil {
			logrus.Error("Error parsing CIDR ", cidrs[0])
		}
		cred.Cidr = cidr
	}

	ip := net.ParseIP(d.Bucket)
	if ip == nil {
		logrus.Errorf("Bucket name is not IP, bug: %s", d.Bucket)
		return nil
	}

	return &CachedInstance{
		Ip:    net.ParseIP(d.Bucket),
		Lport: lport,
		Rport: rport,
		ID:    &cred,
	}
}

func (c *riakConn) GetAllNetID(ip net.IP) ([]CachedInstance, error) {
	query := fmt.Sprintf("%s:%s AND %s:%s", QUERY_BUCKET, ip.String(),
		QUERY_BUCKET_TYPE, RIAK_BUCKET_TYPE)
	cmd, err := riak.NewSearchCommandBuilder().
		WithReturnFields(QUERY_KEY, NETMAP_PID, NETMAP_ID, NETMAP_CIDR,
			NETMAP_TYPE, QUERY_BUCKET).
		WithIndexName(RIAK_INDEX_NAME).
		WithQuery(query).Build()

	result := make([]CachedInstance, 0)
	if err != nil {
		logrus.Debug("error in building fetch all net id map command")
		return result, err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error executing fetch all net id map command")
		return result, err
	}

	if actual, ok := cmd.(*riak.SearchCommand); ok {
		if actual.Response.Docs == nil || len(actual.Response.Docs) == 0 {
			return result, nil
		}
		for _, d := range actual.Response.Docs {
			inst := c.genCachedInstance(d)
			if inst != nil {
				result = append(result, *inst)
			}
		}
		return result, nil
	}
	logrus.Debug("error in reading response of get net id map")
	return result, errors.New("Unknown command")
}

func (c *riakConn) SearchIDNet(uuid string) ([]CachedInstance, error) {
	query := fmt.Sprintf("%s:%s AND %s:%s", NETMAP_ID, uuid,
		QUERY_BUCKET_TYPE, RIAK_BUCKET_TYPE)
	cmd, err := riak.NewSearchCommandBuilder().
		WithReturnFields(QUERY_KEY, NETMAP_PID, NETMAP_ID, NETMAP_CIDR,
			NETMAP_TYPE, QUERY_BUCKET).
		WithIndexName(RIAK_INDEX_NAME).
		WithQuery(query).Build()
	result := make([]CachedInstance, 0)
	if err != nil {
		logrus.Debug("error in building search id to net map command")
		return result, err
	}
	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error executing search ID command")
		return result, err
	}

	if actual, ok := cmd.(*riak.SearchCommand); ok {

		if actual.Response.Docs == nil || len(actual.Response.Docs) == 0 {
			return result, nil
		}
		for _, d := range actual.Response.Docs {
			inst := c.genCachedInstance(d)
			if inst != nil {
				result = append(result, *inst)
			}
		}
		return result, nil
	}
	return result, errors.New("unknown command")
}

func (c *riakConn) PutPodMap(ip net.IP, pod *CachedPod) error {
	return nil
}

// GetPod will delete expired pod.
func (c *riakConn) GetPodMap(ip net.IP) (*CachedPod, error) {
	return nil, nil
}

func (c *riakConn) Shutdown() error {
	if c.Client == nil {
		return errors.New("riak client is not connected")
	}
	return c.Client.Stop()
}
