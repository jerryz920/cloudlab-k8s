package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"proxy/internal/pkg/kvstore"

	logrus "github.com/sirupsen/logrus"
)

type MetadataProxy struct {
	client    *http.Client
	addr      string
	newCaches []Cache
	pods      map[string]Pod
}

func (r MetadataRequest) ByteBuf() (*bytes.Buffer, error) {
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(&r); err != nil {
		return nil, err
	}
	return &buf, nil
}

var (
	ipRangeMatch *regexp.Regexp = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)-(\d+)`)
	ipPortMatch  *regexp.Regexp = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)`)
	pidMatch     *regexp.Regexp = regexp.MustCompile(`\['([-a-zA-Z0-9_]+)'\]`)
	pStore       map[string]*Principal
)

func ParseIP2(msg string) (string, int, int, int) {
	if matches := ipPortMatch.FindStringSubmatch(msg); len(matches) != 3 {
		return "", -1, -1, http.StatusBadRequest
	} else {
		var p1 int64
		var err error
		if p1, err = strconv.ParseInt(matches[2], 10, 32); err != nil {
			return "", 0, 0, http.StatusBadRequest
		}
		return matches[1], int(p1), -1, http.StatusOK
	}

}

func ParseIP3(msg string) (string, int, int, int) {
	if matches := ipRangeMatch.FindStringSubmatch(msg); len(matches) != 4 {
		return ParseIP2(msg)
	} else {
		var p1, p2 int64
		var err error
		if p1, err = strconv.ParseInt(matches[2], 10, 32); err != nil {
			logrus.Errorf("error parsing port min: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		if p2, err = strconv.ParseInt(matches[3], 10, 32); err != nil {
			logrus.Errorf("error parsing port max: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		return matches[1], int(p1), int(p2), http.StatusOK
	}
}

func ParseIPNew(msg string) (net.IP, int, int, int) {
	ipstr, p1, p2, status := ParseIP3(msg)
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, -1, -1, http.StatusBadRequest
	}
	return ip, p1, p2, status
}

func SetCommonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func ReadRequest(r *http.Request) (*MetadataRequest, int) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("error reading the body %v\n", err)
		return nil, http.StatusBadRequest
	}
	logrus.Debugf("request body = %s, url = %s", string(data), r.URL.RequestURI())
	buf := bytes.NewBuffer(data)
	d := json.NewDecoder(buf)
	mr := MetadataRequest{
		targetType: NORMAL_INSTANCE_TYPE,
		targetCidr: nil,
	}
	if err := d.Decode(&mr); err != nil {
		logrus.Errorf("error decoding the body %v\n", err)
		return nil, http.StatusBadRequest
	} else {
		return &mr, http.StatusOK
	}
}

func (c *MetadataProxy) getUrl(api string) string {
	addr := ""
	if !strings.HasSuffix(c.addr, "/") {
		addr += c.addr + "/"
	}
	if !strings.HasPrefix(c.addr, "http://") {
		addr = "http://" + addr
	}
	if strings.HasPrefix(api, "/") {
		api = api[1:]
	}
	return addr + api
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metadata Service Proxy"))
}

type Addresses []string

//// Configurations
var (
	RiakAddrs  = make(Addresses, 0)
	ConfDebug  bool
	SafeAddr   string
	ListenAddr string
	Nworker    int
)

func (s *Addresses) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *Addresses) Set(val string) error {
	_, err := net.ResolveTCPAddr("tcp", val)
	if err != nil {
		logrus.Info("error parsing tcp address: ", val)
		return err
	}
	*s = append(*s, val)
	return nil
}

func config() {
	flag.Var(&RiakAddrs, "addr", "riak addresses")
	flag.BoolVar(&ConfDebug, "debug", false, "set debug output")
	flag.StringVar(&SafeAddr, "safe", "localhost:7777", "set safe address")
	flag.StringVar(&ListenAddr, "listen", "0.0.0.0:19851", "listen address")
	/// Each worker has its own cache
	flag.IntVar(&Nworker, "nworker", 64, "num of workers")
	flag.Parse()
}

func main() {
	config()

	if ConfDebug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	formatter := new(logrus.TextFormatter)
	formatter.DisableLevelTruncation = false
	formatter.FullTimestamp = false
	formatter.TimestampFormat = "1 2 3:4:5.999999"
	logrus.SetFormatter(formatter)

	if len(RiakAddrs) == 0 {
		RiakAddrs = []string{"localhost:8087"}
	}
	logrus.Infof("Riakaddresses : %v", RiakAddrs)

	riakClient := NewRiakConn()
	if err := riakClient.Connect(RiakAddrs); err != nil {
		logrus.Errorf("can not connect to riak address: %v, %s", RiakAddrs, err)
		os.Exit(1)
	}
	logrus.Info("Riak connected! Starting the API server")

	caches := make([]Cache, Nworker)
	for i := 0; i < Nworker; i++ {
		caches[i] = NewCache(riakClient)
	}

	client := MetadataProxy{
		client: &http.Client{
			Transport: &http.Transport{
				DisableCompression:  true,
				MaxIdleConnsPerHost: 256,
			},
			Timeout: time.Second * 15,
		},
		addr:      SafeAddr,
		newCaches: caches,
		pods:      make(map[string]Pod),
	}
	server := kvstore.NewKvStore(rootHandler)

	SetupNewAPIs(&client, server)
	//// New APIs
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	if err := server.ListenAndServe(ListenAddr); err != nil {
		logrus.Fatal("can not listen on address: ", err)
	}
}
