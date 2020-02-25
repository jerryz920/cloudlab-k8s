package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/biogo/store/interval"
	"github.com/sirupsen/logrus"
)

const (
	ForceLookup = uintptr(0)
)

/// defines the principal map for lookup
type Index struct {
	Pmin, Pmax int
	Id         uintptr
}

type PrincipalIndex struct {
	Index
	P         string
	PP        string /// parent pid
	GroupPort int
	GroupP    string
}

func (p Index) Overlap(b interval.IntRange) bool {
	return p.Pmax > b.Start && p.Pmin < b.End
}
func (p Index) ID() uintptr { return p.Id }
func (p Index) Range() interval.IntRange {
	return interval.IntRange{p.Pmin, p.Pmax}
}
func (p Index) String() string {
	return fmt.Sprintf("[%d,%d)#%d", p.Pmin, p.Pmax, p.Id)
}

type Pmap struct {
	Identities map[string]*interval.IntTree // ip -> port range map
	PidMap     map[string]*CachedInstance   // uuid -> ip, port range, and ppid
	LockMap    map[string]*sync.Mutex
	counter    int
}

func NewPmap() *Pmap {
	return &Pmap{
		Identities: make(map[string]*interval.IntTree),
		PidMap:     make(map[string]*CachedInstance),
		counter:    1,
	}
}

/// NOTE THIS ONLY WORKS ON 64BIT MACHINE!
func ComputeID(ip string, p1 int, p2 int) uintptr {
	v0 := binary.BigEndian.Uint32(net.ParseIP(ip).To4())
	v1 := (uint64(v0)) << 32
	v2 := uint64(p1) << 16
	v3 := uint64(p2)
	return uintptr(v1 + v2 + v3)
}

func (m *Pmap) Loaded(ip string) bool {
	_, ok := m.Identities[ip]
	return ok
}

func (m *Pmap) Unload(ip string) {
	delete(m.Identities, ip)
	/// removing the identity mapped uuids
}

func (m *Pmap) CreatePrincipal(ip net.IP, pmin int, pmax int, p string) {
	m.CreatePrincipalPP(ip, pmin, pmax, p, "", nil, "")
}

/// Need to refactor all these shits, the type is disgusting, p, pp, ppppp
func (m *Pmap) CreatePrincipalPP(ip net.IP, pmin int, pmax int, p string, pp string,
	cidr *net.IPNet, t string) {
	m.counter++
	ipstr := ip.String()
	index := PrincipalIndex{
		Index: Index{
			Id:   ComputeID(ipstr, pmin, pmax+1),
			Pmin: pmin,
			Pmax: pmax + 1,
		},
		P:  p,
		PP: pp,
	}
	if tree, ok := m.Identities[ipstr]; ok {
		tree.Insert(&index, false)
	} else {
		m.Identities[ipstr] = &interval.IntTree{}
		m.Identities[ipstr].Insert(&index, false)
	}
	m.PidMap[p] = &CachedInstance{
		Ip:    ip,
		Lport: pmin,
		Rport: pmax,
		ID: &InstanceCred{
			Pid:  p,
			PPid: pp,
			Type: t,
			Cidr: cidr,
		},
	}
}

func (m *Pmap) SetPrincipalGroupPort(ip string, port int, P string) {
	index, err := m.GetIndex(ip, port)
	if index == nil || err != nil {
		logrus.Error("Principal Index not found: ", err)
		return
	}
	index.GroupPort = port
	index.GroupP = P
}

func (m *Pmap) DeletePrincipal(ip string, pmin int, pmax int) error {
	if tree, ok := m.Identities[ip]; ok {
		return tree.Delete(&Index{pmin, pmax + 1, ComputeID(ip, pmin, pmax+1)}, false)
	} else {
		logrus.Errorf("Principal to delete not found: %s:%d-%d", ip, pmin, pmax)
		return errors.New("not found")
	}
}
func (m *Pmap) GetIndex(ip string, port int) (*PrincipalIndex, error) {

	if tree, ok := m.Identities[ip]; ok {
		indexes := tree.Get(&Index{
			Pmin: port,
			Pmax: port + 1,
			Id:   ForceLookup,
		})
		if len(indexes) == 0 {
			return nil, nil
		}
		/// find the inner most one
		found := indexes[0]
		logrus.Debugf("debug: found index: %v, %d %d\n", found.ID(), found.Range().Start, found.Range().End)
		for i := 1; i < len(indexes); i++ {
			fmt.Printf("debug: found index: %v, %d %d\n", indexes[i].ID(), indexes[i].Range().Start, indexes[i].Range().End)
			if found.Range().Start <= indexes[i].Range().Start &&
				found.Range().End >= indexes[i].Range().End {
				found = indexes[i]
			}
		}
		pindex, ok := found.(*PrincipalIndex)
		if !ok {
			logrus.Debugf("type conversion error, required %T, actual %T",
				PrincipalIndex{}, found)
			return nil, errors.New("type conversion error")
		}
		if pindex.Pmin <= port && pindex.Pmax > port {
			return pindex, nil
		}
		logrus.Debugf("Not found: %s:%d", ip, port)
		return nil, nil
	}
	return nil, nil
}

func (m *Pmap) GetPrincipal(ip string, port int) (string, error) {
	index, err := m.GetIndex(ip, port)
	if index != nil {
		return index.P, nil
	} else {
		return "", err
	}
}

func (m *Pmap) GetCachedInstance(uuid string) (*CachedInstance, error) {
	if inst, ok := m.PidMap[uuid]; ok {
		return inst, nil
	}
	return nil, errors.New("Not Found")
}

func (m *Pmap) PutCachedInstance(inst *CachedInstance) {
	logrus.Debugf("Caching instance: %s, %d, %d, %v", inst.Ip, inst.Lport, inst.Rport, *inst.ID)
	m.CreatePrincipalPP(inst.Ip, inst.Lport, inst.Rport,
		inst.ID.Pid, inst.ID.PPid, inst.ID.Cidr, inst.ID.Type)
	m.PidMap[inst.ID.Pid] = inst
}

func (m *Pmap) DelCachedInstance(inst *CachedInstance) {
	m.DelCachedInstanceAlt(inst.Ip, inst.Lport, inst.Rport, inst.ID.Pid)
}

func (m *Pmap) DelCachedInstanceAlt(ip net.IP, lport, rport int, pid string) {
	logrus.Debugf("Deleting instance: %v %d %d %s", ip, lport, rport, pid)
	ipstr := ip.String()
	m.DeletePrincipal(ipstr, lport, rport)
	delete(m.PidMap, pid)
}
