package main

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	jhttp "github.com/jerryz920/utils/goutils/http"
	"github.com/sirupsen/logrus"
)

const (
	IaaSProvider = "152.3.145.38:444"
)

func NotImplemented(w http.ResponseWriter, r *http.Request) {
	logrus.Error("Method not implemented!")
}

func (c *MetadataProxy) getCache(remoteAddr string, auth string) Cache {
	hasher := fnv.New32()
	hasher.Write([]byte(remoteAddr))
	idx := hasher.Sum32() % (uint32)(len(c.newCaches))
	hasher = fnv.New32()
	hasher.Write([]byte(auth))
	idx1 := hasher.Sum32() % (uint32)(len(c.newCaches))
	logrus.Infof("cache %s to %d, %s to %d", remoteAddr, idx, auth, idx1)
	return c.newCaches[idx1]
}

func (c *MetadataProxy) authenticate(mr *MetadataRequest, principal string) (*CachedInstance, int) {
	if strings.HasPrefix(principal, NOAUTH_PREFIX) {
		logrus.Info("noauth header found!")
		return nil, http.StatusUnauthorized
	}

	ip, lport, _, ok := ParseIPNew(principal)
	if ok != http.StatusOK {
		logrus.Debug("Principal looks like an UUID")
		/// Try UUID
		cachedInstance, status := mr.cache.GetInstanceFromID(principal)
		if status != http.StatusOK {
			logrus.Error("fail to authenticate the UUID")
			return nil, status
		}
		return &cachedInstance, status
	} else {
		cachedInstance, status := mr.cache.GetInstanceFromNetMap(ip, lport)
		if status != http.StatusOK {
			logrus.Error("fail to authenticate the network address")
			return nil, status
		}
		return &cachedInstance, status
	}
}

func (c *MetadataProxy) newAuth(r *http.Request, authPrincipal bool) (*MetadataRequest, int) {
	mr, status := ReadRequest(r)
	mr.method = r.Method
	if mr.Principal == "" {
		mr.Principal = r.RemoteAddr
	}
	mr.url = r.URL.RequestURI()

	if status != http.StatusOK {
		logrus.Error("error reading request in newAuth")
		return nil, status
	}

	mr.cache = c.getCache(r.RemoteAddr, mr.Auth)
	mr.Auth = "" //clear it for compatibility
	// Compute cache index to use for this connection. We are assuming the
	// same IP and port will behave like a VM

	if mr.Principal == IaaSProvider {
		mr.Principal = IaaSProvider
		return mr, status
	}

	cachedInstance, status := c.authenticate(mr, mr.Principal)
	if status != http.StatusOK {
		if authPrincipal {
			logrus.Error("fail to authenticate the principal field")
			return nil, status
		} else {
			logrus.Info("principal not authenticated, proceed as external principal")
			return mr, http.StatusOK
		}
	}

	remoteIp, remotePort, _, status := ParseIPNew(r.RemoteAddr)
	if status != http.StatusOK {
		logrus.Error("fail to parse remote address in request: Must be a bug: ", r.RemoteAddr)
		return nil, status
	}
	if remoteIp.Equal(cachedInstance.Ip) || remotePort < cachedInstance.Lport ||
		remotePort > cachedInstance.Rport {
		////Not rejecting the request for experiments.
		logrus.Error("incoming port is not within the range.")
	}

	mr.Principal = cachedInstance.ID.Pid
	mr.ParentBear = cachedInstance.ID.PPid
	mr.ip = cachedInstance.Ip
	mr.lport = cachedInstance.Lport
	mr.rport = cachedInstance.Rport
	mr.instance = cachedInstance
	return mr, http.StatusOK
}

//// authorize if a control message will be sent by the metadata service
/// the targetIp, targetLport, targetRport is to be checked
/// the allowUUID field allows removing instances using its UUID directly
func (c *MetadataProxy) authzControl(mr *MetadataRequest, targetAddr string, allowUUID bool) int {
	var ok int
	/// Still parse the address
	mr.targetIp, mr.targetLport, mr.targetRport, ok = ParseIPNew(targetAddr)

	if ok != http.StatusOK {
		if allowUUID {
			logrus.Infof("Can not parse target field as address, trying UUID")
			cachedInstance, status := mr.cache.GetInstanceFromID(targetAddr)
			if status != http.StatusOK {
				logrus.Error("fail to authenticate the UUID")
				return status
			}
			mr.targetIp, mr.targetLport, mr.targetRport =
				cachedInstance.Ip, cachedInstance.Lport, cachedInstance.Rport
		} else {
			logrus.Errorf("error parsing target IP %s", targetAddr)
			return ok
		}
	}

	if mr.Principal == IaaSProvider {
		return http.StatusOK
	}

	if mr.ip.Equal(mr.targetIp) && mr.lport <= mr.targetLport && mr.rport >= mr.targetRport {
		return http.StatusOK
	}
	// last resort: check if the ip is inside some allocation
	if alloc := mr.instance.ID.Cidr; mr.instance.ID.Type == VM_INSTANCE_TYPE && alloc != nil {
		if alloc.Contains(mr.targetIp) {
			return http.StatusOK
		}
	}
	return http.StatusUnauthorized
}

func (c *MetadataProxy) workaroundEmptyTrustHub(mr *MetadataRequest) {
	/// Creating VM instance
	if len(mr.OtherValues) == 4 && mr.Principal == IaaSProvider {
		logrus.Warn("Workaround VM creation: ", mr.OtherValues)
		mr.OtherValues = append(mr.OtherValues, DEFAULT_VM_TRUST_HUB)
	} else if len(mr.OtherValues) == 3 {
		logrus.Warn("Workaround Instance creation", mr.OtherValues)
		mr.OtherValues = append(mr.OtherValues, DEFAULT_CTN_TRUST_HUB)
	}
}

func (c *MetadataProxy) preInstanceCallHandler(mr *MetadataRequest) (string, int) {
	targetAddrIndex := 2
	if mr.targetAddrIndex > 0 {
		targetAddrIndex = mr.targetAddrIndex
	}
	if status := c.authzControl(mr, mr.OtherValues[targetAddrIndex], false); status != http.StatusOK {
		return fmt.Sprintf("can not authorize request: %s, %s:%d-%d to %s:%d-%d\n",
				mr.Principal, mr.ip, mr.lport, mr.rport,
				mr.targetIp, mr.targetLport, mr.targetRport),
			status
	}

	return "", http.StatusOK
}

func (c *MetadataProxy) postInstanceCreationHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	if status != http.StatusOK {
		return string(data), status
	}

	instance := &CachedInstance{
		Ip:    mr.targetIp,
		Lport: mr.targetLport,
		Rport: mr.targetRport,
		ID: &InstanceCred{
			Pid:  mr.OtherValues[0],
			PPid: mr.Principal,
			Type: mr.targetType,
			Cidr: mr.targetCidr,
		},
	}
	mr.cache.PutInstance(instance)
	return fmt.Sprintf("{\"message\": \"['%s']\"}\n", mr.OtherValues[0]),
		http.StatusOK
}

func (c *MetadataProxy) postInstanceDeletionHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	if status != http.StatusOK {
		return string(data), status
	}

	mr.cache.DelInstance(mr.targetIp, mr.targetLport, mr.targetRport, mr.OtherValues[0])
	return fmt.Sprintf("{\"message\": \"['%s']\"}\n", mr.OtherValues[0]),
		http.StatusOK
}

func (c *MetadataProxy) preVMInstanceCallHandler(mr *MetadataRequest) (string, int) {
	if mr.Principal != IaaSProvider {
		return "only IaaS provider can create VM", http.StatusUnauthorized
	}
	// Some unexpected changes in upstream repo.
	_, cidr, err := net.ParseCIDR(mr.OtherValues[4])
	if err != nil {
		msg := fmt.Sprintf("can not allocate CIDR %s\n", err)
		return msg, http.StatusBadRequest
	}
	mr.targetCidr = cidr
	mr.targetType = VM_INSTANCE_TYPE
	mr.targetAddrIndex = 3
	/// Cidr is no longer used

	return c.preInstanceCallHandler(mr)
}

func (c *MetadataProxy) preLegacyInstanceCallHandler(mr *MetadataRequest) (string, int) {

	if mr.Principal == IaaSProvider {
		// Creating VM instance
		msg := fmt.Sprintf("Should not use legacy API to create VMs!")
		return msg, http.StatusBadRequest
	}

	newRequest := make([]string, 2)
	newRequest[0] = mr.OtherValues[0]
	newRequest[1] = mr.OtherValues[1]
	mr.OtherValues = newRequest
	return c.preInstanceCallHandler(mr)
}

func (c *MetadataProxy) createInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createInstanceLegacy(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preLegacyInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createVMInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preVMInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) deleteInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preInstanceCallHandler,
		c.postInstanceDeletionHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) deleteVMInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preVMInstanceCallHandler,
		c.postInstanceDeletionHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createEndorsementLink(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		creatorInstance, status := c.authenticate(mr, mr.OtherValues[0])
		if status != http.StatusOK {
			logrus.Info("Can not authenticate endorser as instance, proceed as external principal")
		} else {
			mr.OtherValues[0] = creatorInstance.ID.Pid
		}
		return "", http.StatusOK
	}, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createEndorsement(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {

		// authenticated as instance, replace the endorsement method
		// /postEndorsement -> /postInstanceEndorsement
		if mr.instance != nil {
			mr.url = strings.Replace(mr.url, "/post", "/postInstance", 1)
		}
		// not authenticated

		// We do not want the power of instance endorsement so far.
		//creatorInstance, status := c.authenticate(mr.OtherValues[0])
		//if status != http.StatusOK {
		//	logrus.Info("Can not authenticate endorser as instance, proceed as external principal")
		//} else {
		//	mr.OtherValues[0] = creatorInstance.ID.Pid
		//}
		return "", http.StatusOK
	}, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) lazyDeleteInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r,
		func(mr *MetadataRequest) (string, int) {
			if status := c.authzControl(mr, mr.OtherValues[0], true); status != http.StatusOK {
				return fmt.Sprintf("can not authorize request: %s, %s:%d-%d to %s:%d-%d\n",
						mr.Principal, mr.ip, mr.lport, mr.rport,
						mr.targetIp, mr.targetLport, mr.targetRport),
					status
			}
			mr.targetType = NORMAL_INSTANCE_TYPE
			mr.targetCidr = nil
			return "", http.StatusOK
		},
		c.postInstanceDeletionHandler,
		true,
	)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createMembership(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r,
		func(mr *MetadataRequest) (string, int) {
			memberInstance, status := c.authenticate(mr, mr.OtherValues[1])
			if status != http.StatusOK {
				return fmt.Sprintf("cannot authenticate membership creator\n"), status
			}
			mr.OtherValues[1] = memberInstance.ID.Pid
			return "", http.StatusOK
		},
		nil, true,
	)

	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createSelfConfig(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		mr.OtherValues = append([]string{mr.Principal}, mr.OtherValues...)
		// FIXME: a lot of redundancy.

		remain := len(mr.OtherValues) - 1
		current := 1 /// skip the instance ID.
		for remain > 10 {
			args := make([]string, 0)
			args = append(args, mr.OtherValues[0])
			args = append(args, mr.OtherValues[current:current+10]...)

			copyr := &MetadataRequest{
				Principal:   mr.Principal,
				Auth:        mr.Auth,
				OtherValues: args,
				method:      "POST",
				url:         "/postInstanceConfig5",
				instance:    mr.instance,
				cache:       mr.cache,
			}
			msg, status := c.newHandler(copyr, nil, nil)

			if status != http.StatusOK {
				logrus.Error("error posting configurations, original: %v, left: %v",
					mr.OtherValues, mr.OtherValues[current:])
				return msg, status
			}
			current += 10
			remain -= 10
		}
		mr.OtherValues = append(mr.OtherValues[:1], mr.OtherValues[current:]...)
		mr.method = "POST"
		fmt.Println("othervalues=%v", mr.OtherValues)
		mr.url = fmt.Sprintf("%s%d", "/postInstanceConfig", remain/2)
		/// The handler will continue to handle the remaining configs
		return "", http.StatusOK
	}, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createInstanceConfig(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {

		cachedInstance, status := c.authenticate(mr, mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("Fail to authenticate the instance %s",
				mr.OtherValues[0]), status
		}
		if mr.Principal != cachedInstance.ID.PPid {
			return fmt.Sprintf("Config can only be published by host instance:"+
					"Speaker: %s, Pid: %s, Saved-PPid: %s", mr.Principal,
					cachedInstance.ID.Pid, cachedInstance.ID.PPid),
				http.StatusUnauthorized
		}
		mr.OtherValues[0] = cachedInstance.ID.Pid

		remain := len(mr.OtherValues) - 1
		current := 1 /// skip the instance ID.
		for remain > 10 {
			args := make([]string, 0)
			args = append(args, mr.OtherValues[0])
			args = append(args, mr.OtherValues[current:current+10]...)

			copyr := &MetadataRequest{
				Principal:   mr.Principal,
				Auth:        mr.Auth,
				OtherValues: args,
				method:      "POST",
				url:         "/postInstanceConfig5",
				instance:    mr.instance,
				cache:       mr.cache,
			}
			msg, status := c.newHandler(copyr, nil, nil)

			if status != http.StatusOK {
				logrus.Error("error posting configurations, original: %v, left: %v",
					mr.OtherValues, mr.OtherValues[current:])
				return msg, status
			}
			current += 10
			remain -= 10
		}
		mr.OtherValues = append(mr.OtherValues[:1], mr.OtherValues[current:]...)
		mr.method = "POST"
		mr.url = fmt.Sprintf("%s%d", mr.url, remain/2)
		/// The handler will continue to handle the remaining configs
		return "", http.StatusOK
	}, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createInstanceKvConfig(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {

		cachedInstance, status := c.authenticate(mr, mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("Fail to authenticate the instance %s",
				mr.OtherValues[0]), status
		}
		if mr.Principal != cachedInstance.ID.PPid {
			return fmt.Sprintf("Config can only be published by host instance:"+
					"Speaker: %s, Pid: %s, Saved-PPid: %s", mr.Principal,
					cachedInstance.ID.Pid, cachedInstance.ID.PPid),
				http.StatusUnauthorized
		}
		mr.OtherValues[0] = cachedInstance.ID.Pid

		remain := len(mr.OtherValues) - 1
		// There should be odd numbers of arguments left, which forms key value list
		if remain%2 == 0 || remain < 2 {
			return fmt.Sprintf("Config must be key value list:"+
					"Speaker: %s, Pid: %s, Saved-PPid: %s", mr.Principal,
					cachedInstance.ID.Pid, cachedInstance.ID.PPid),
				http.StatusBadRequest
		}

		result := []string{}
		for current := 2; current < len(mr.OtherValues); current += 2 {
			key, val := mr.OtherValues[current], mr.OtherValues[current+1]
			result = append(result, fmt.Sprintf("[\\\"%s\\\", \\\"%s\\\"]", key, val))
		}

		actualArgs := append(mr.OtherValues[:2], fmt.Sprintf("\"[%s]\"", strings.Join(result, ",")))

		copyr := &MetadataRequest{
			Principal:   mr.Principal,
			Auth:        mr.Auth,
			OtherValues: actualArgs,
			method:      "POST",
			url:         "/postInstanceConfig1",
			instance:    mr.instance,
			cache:       mr.cache,
		}
		msg, status := c.newHandler(copyr, nil, nil)

		if status != http.StatusOK {
			logrus.Error("error posting configurations, original: %v", mr.OtherValues)
			return msg, status
		}
		return "", http.StatusOK
	}, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))

}

func (c *MetadataProxy) handleCheckInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	logrus.Info("new request -> ", r.URL.RequestURI())
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		/// convert otherValues[0] to uuid
		if mr.OtherValues[0] == IaaSProvider {
			return "", http.StatusOK
		}

		cache, status := c.authenticate(mr, mr.OtherValues[0])
		if status != http.StatusOK {
			// Proceed as object
			mr.ParentBear = "no-such-pid"
			logrus.Info("target not found, proceed as non-instance object")
			return "", http.StatusOK
		}
		mr.OtherValues[0] = cache.ID.Pid
		logrus.Info("PPid=", cache.ID.PPid)
		mr.ParentBear = cache.ID.PPid
		return "", http.StatusOK
	}, nil, false)
	logrus.Info("end of request: ", msg, " ", status)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) handleOther(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, nil, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

// Does not authenticate the speaker. This means we don't trust it. It can create
// link that points to wrong content. But as long as the end endorsements can be
// authenticated, it is all right.
func (c *MetadataProxy) handleOtherNoAuth(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, nil, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) newHandlerUnwrapped(r *http.Request, preHook func(*MetadataRequest) (string, int),
	postHook func(*MetadataRequest, []byte, int) (string, int), authPrincipal bool) (string, int) {
	/// FIXME: authparent is no longer used
	metareq, status := c.newAuth(r, authPrincipal)
	if status != http.StatusOK {
		return "can not authenticate request\n", status
	}
	return c.newHandler(metareq, preHook, postHook)
}

/// A pre hook will be run before sending out request, post hook will run after
// correctly fetch the data
func (c *MetadataProxy) newHandler(mr *MetadataRequest, preHook func(*MetadataRequest) (string, int),
	postHook func(*MetadataRequest, []byte, int) (string, int)) (string, int) {

	t1 := time.Now()
	logrus.Info("New handler: ")
	if preHook != nil {
		msg, status := preHook(mr)
		if status != http.StatusOK {
			return msg, status
		}
	}
	t2 := time.Now()

	buf, err := EncodingMetadataRequest(mr)
	if err != nil {
		msg := fmt.Sprintf("error encoding mr %v\n", err)
		return msg, http.StatusInternalServerError
	}
	logrus.Infof("out going req=[%s]", buf.String())

	outreq, err := http.NewRequest(mr.method, c.getUrl(mr.url), buf)
	if err != nil {
		msg := fmt.Sprintf("fail to generate new request %v\n", err)
		return msg, http.StatusInternalServerError
	}

	resp, err := c.client.Do(outreq)
	if err != nil {
		msg := fmt.Sprintf("error proxying post instance set % v\n", err)
		if resp == nil {
			return msg, http.StatusInternalServerError
		} else {
			return msg, resp.StatusCode
		}
	}

	data, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		msg := fmt.Sprintf("error reading the response from server: %v\n", err)
		return msg, http.StatusInternalServerError
	}
	logrus.Infof("SAFE Response %s: %s", mr.url, string(data))
	t3 := time.Now()

	if postHook != nil {
		msg, status := postHook(mr, data, resp.StatusCode)
		t4 := time.Now()
		logrus.Infof("PERF %s %f %f %f", mr.url, t2.Sub(t1).Seconds(), t3.Sub(t2).Seconds(), t4.Sub(t3).Seconds())
		return msg, status
	}
	logrus.Infof("PERF %s %f %f", mr.url, t2.Sub(t1).Seconds(), t3.Sub(t2).Seconds())
	// We allow call another http method in posthook, however, its result
	// is consumed internally only
	return string(data), resp.StatusCode
}

func (c *MetadataProxy) authenticate_addr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	saddr := vars["instance_addr"]
	logrus.Infof("authenticating addr: %s", saddr)
	cache := c.getCache(saddr, "1")
	addr, port, _, status := ParseIP2(saddr)
	netaddr := net.ParseIP(addr)
	if status != http.StatusOK || netaddr == nil {
		logrus.Errorf("must provide a valid ip address: %v", saddr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cachedInstance, status := cache.GetInstanceFromNetMap(netaddr, port)
	if status != http.StatusOK {
		logrus.Error("fail to authenticate the network address")
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s\n%s\n", cachedInstance.ID.Pid, cachedInstance.ID.PPid)))
	}

}

func SetupNewAPIs(c *MetadataProxy, server *jhttp.APIServer) {

	server.AddRoute("/postInstance", c.createInstance, "")
	server.AddRoute("/postInstanceSet", c.createInstanceLegacy, "")
	server.AddRoute("/retractInstanceSet", c.deleteInstance, "")
	server.AddRoute("/postVMInstance", c.createVMInstance, "")
	server.AddRoute("/delInstance", c.deleteInstance, "")
	server.AddRoute("/delVMInstance", c.deleteVMInstance, "")
	server.AddRoute("/lazyDeleteInstance", c.lazyDeleteInstance, "")
	/// Handling any num of configs
	server.AddRoute("/postInstanceConfig", c.createInstanceConfig, "")
	server.AddRoute("/postInstanceKV", c.createInstanceKvConfig, "")
	server.AddRoute("/postSelfConfig", c.createSelfConfig, "")

	/// The proxy to do here is similar, just authenticate the IDs
	server.AddRoute("/postAckMembership", c.createMembership, "")
	server.AddRoute("/postMembership", c.createMembership, "")
	/// No need to authenticate the principal, as only linking
	/// inside will be used.
	server.AddRoute("/postEndorsementLink", c.createEndorsementLink, "")
	server.AddRoute("/postEndorsement", c.createEndorsement, "")
	server.AddRoute("/postConditionalEndorsement", c.createEndorsement, "")
	server.AddRoute("/postParameterizedEndorsement", c.createEndorsement, "")

	otherMethods := []string{
		"/postCluster",
		"/delAckMembership",
		"/delCluster",
		"/delConditionalEndorsement",
		"/delEndorsement",
		"/delInstanceAuthID",
		"/delInstanceAuthKey",
		"/delInstanceCidrConfig",
		"/delInstanceConfig1",
		"/delInstanceConfig2",
		"/delInstanceConfig3",
		"/delInstanceConfig4",
		"/delInstanceConfig5",
		"/delInstanceControl",
		"/delLinkImageOwner",
		"/delMembership",
		"/delParameterizedConnection",
		"/delParameterizedEndorsement",
		"/delVpcConfig1",
		"/delVpcConfig2",
		"/delVpcConfig3",
		"/delVpcConfig4",
		"/delVpcConfig5",
		"/postConditionalEndorsement",
		//"/postInstanceAuthID",
		//"/postInstanceAuthKey",
		//"/postInstanceCidrConfig",
		"/postInstanceConfigList",
		"/postInstanceConfig1",
		"/postInstanceConfig2",
		"/postInstanceConfig3",
		"/postInstanceConfig4",
		"/postInstanceConfig5",
		//"/postInstanceControl",
		"/postParameterizedConnection",
		"/postParameterizedEndorsement",
		"/postVpcConfig1",
		"/postVpcConfig2",
		"/postVpcConfig3",
		"/postVpcConfig4",
		"/postVpcConfig5",
	}

	checkMethods := []string{
		"/checkClusterGuard",
		"/checkCodeQuality",
		"/checkContainerIsolation",
		"/checkFetch",
		"/checkAttester",
		"/checkBuilder",
		"/checkProperty",
		"/checkLaunches",
		"/checkBuildsFrom",
		"/checkEndorse",
		"/checkTrustedCluster",
		"/checkMySQLConnection",
		"/checkKeystoneReq",
		"/checkAllForSparkEvaluation",
		//	"/checkTrustedCode", Should use some other check
		"/checkMaster",
		"/checkWorkerFromMaster",
		"/checkExecutorFromDriver",
		"/checkDriverFromHDFS",
		"/checkTrustedConnections",
		"/checkK8sHDFSAccess",
		"/checkSafeSpark",
		"/checkK8sVm",
		"/checkTrustedEndorser",
		"/checkEndorsed",
		"/checkPodAttestation",
		"/debugCheck1",
		"/debugCheck2",
		"/debugCheck3",
		"/debugCheck4",
		"/debugCheck5",
		"/debugCheck6",
		"/debugCheck7",
		"/debugCheck8",
		"/checkPodByPolicy",
		"/checkHasConfig",
		"/debug",
	}

	noauthMethods := []string{
		"/postTrustedEndorser",
		"/postTrustHubLink",
		"/postImageSpec",
		"/postPropertySpec",
		"/postImagePolicy",
		"/postProhibitedKeyPolicy",
		"/postRequiredKeyPolicy",
		"/postQualifierKeyPolicy",
	}

	for _, method := range checkMethods {
		server.AddRoute(method, c.handleCheckInstance, "")
	}
	for _, method := range otherMethods {
		server.AddRoute(method, c.handleOther, "")
	}
	for _, method := range noauthMethods {
		server.AddRoute(method, c.handleOtherNoAuth, "")
	}

	server.AddRoute("/authenticate/{instance_addr}", c.authenticate_addr, "")

	return

}
