package kvstore

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// File backed storage and sync

type SimpleStore interface {
	Put(path string, data string)
	Get(path string) string
	Del(path string)
	Keys() []string
}

type ContentStore interface {
	GetKey(value string) []string
	PutValues(key string, values []string)
	GetValues(key string) []string
}

type WatchedStore interface {
	Changed() bool
	Save(path string) error
	Restore(path string, buildIndex bool) error
}

type Store interface {
	SimpleStore
	ContentStore
	WatchedStore
}

type SStore interface {
	SimpleStore
	WatchedStore
}

var (
	watchedSimpleStores map[string]WatchedStore = make(map[string]WatchedStore)
	simpleStoreLock     sync.Mutex
	initFlag            sync.Once
	storeInitialized    bool      = false
	done                chan bool = make(chan bool)
	storeDir            string    = "/var/lib/store" // default dir
)

const (
	separator = ";"
)

type store struct {
	data       map[string][]string
	index      map[string][]string
	lock       sync.Mutex
	buildIndex bool
	changed    bool
}

func startWatcher() {

	go func() {
		exit := false
		for !exit {
			select {
			case <-done:
				exit = true
			case <-time.After(time.Second * time.Duration(writeTimeout)):
			}
			simpleStoreLock.Lock()
			for name, s := range watchedSimpleStores {
				if s.Changed() {
					s.Save(filepath.Join(storeDir, name))
				}
			}
			simpleStoreLock.Unlock()
		}
	}()
}

/// Initialize the store persistent dir
func doInitStore() {
	if storeInitialized {
		return
	}
	err := os.MkdirAll(storeDir, 0755)
	if err != nil {
		logrus.Fatal("can not create store path!", err)
	}
	startWatcher()
	storeInitialized = true
}

func InitStore(path string) {
	storeDir = path
	initFlag.Do(doInitStore)
}

func WrapName(name string, buildIndex bool) string {
	if buildIndex {
		return name
	} else {
		return name + ".NoIndex"
	}
}

func IndexedStore(name string) bool {
	return !strings.HasSuffix(name, ".NoIndex")
}

func NewStore(name string, buildIndex bool) (s Store, err error) {
	name = WrapName(name, buildIndex)
	initFlag.Do(doInitStore)
	simpleStoreLock.Lock()
	if repo, ok := watchedSimpleStores[name]; ok {
		s, ok = repo.(Store)
		if !ok {
			err = errors.New("name exist, but not implemented Store interface")
		} else {
			err = nil
		}

	} else {
		sinner := store{
			data:       make(map[string][]string),
			index:      make(map[string][]string),
			changed:    true, // need write for init
			buildIndex: buildIndex,
		}
		watchedSimpleStores[name] = &sinner
		s = &sinner
		err = nil
	}
	simpleStoreLock.Unlock()
	return
}

func GetStore(name string) (s Store) {
	var ok bool
	initFlag.Do(doInitStore)
	simpleStoreLock.Lock()
	repo, ok := watchedSimpleStores[WrapName(name, true)]
	if !ok {
		s = nil
		repo, ok = watchedSimpleStores[WrapName(name, false)]

	}
	if ok {
		s, ok = repo.(Store)
		if !ok {
			logrus.Error("Found store object, but not implemented Store interface")
			s = nil
		}

	}
	simpleStoreLock.Unlock()
	return
}

func RestartStore(_ bool) error {
	initFlag.Do(doInitStore)
	simpleStoreLock.Lock()
	err := filepath.Walk(storeDir, func(path string, _ os.FileInfo, _ error) error {
		var s store
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		var errInner error
		if IndexedStore(path) {
			errInner = s.Restore(path, true)
		} else {
			errInner = s.Restore(path, false)
		}
		if errInner == nil {
			base := filepath.Base(path)
			watchedSimpleStores[base] = &s
		}
		return errInner
	})
	simpleStoreLock.Unlock()
	return err
}

func GracefulShutdown() {
	done <- true
}

// Assume lock hold!
func (s *store) removeOneIndex(path string, data string) {
	if s.buildIndex {
		if indexes, ok := s.index[data]; ok {
			for i, key := range indexes {
				if key == path {
					s.index[data] = append(s.index[data][:i], s.index[data][i+1:]...)
					break
				}
			}
		}
	}
}

func (s *store) removeIndexes(path string) {
	if s.buildIndex {
		if dataItems, ok := s.data[path]; ok {
			for _, item := range dataItems {
				s.removeOneIndex(path, item)
			}
		}
	}
}

func (s *store) Put(path string, data string) {
	s.lock.Lock()
	s.removeIndexes(path)
	s.data[path] = []string{data}
	if s.buildIndex {
		s.index[data] = append(s.index[data], path)
	}
	s.changed = true
	s.lock.Unlock()
}

func (s *store) Get(path string) string {
	var value string
	s.lock.Lock()
	if values, ok := s.data[path]; ok {
		if len(values) > 0 {
			value = values[0]
		} else {
			value = ""
		}
	} else {
		value = ""
	}
	s.lock.Unlock()
	return value
}

func (s *store) Del(path string) {
	s.lock.Lock()
	s.removeIndexes(path)
	delete(s.data, path)
	s.lock.Unlock()
}

func (s *store) Keys() []string {
	result := make([]string, 0, len(s.data))
	s.lock.Lock()
	for k, _ := range s.data {
		result = append(result, k)
	}
	s.lock.Unlock()
	return result
}

func (s *store) GetKey(value string) (keys []string) {

	var ok bool
	if s.buildIndex {
		s.lock.Lock()
		if keys, ok = s.index[value]; !ok {
			keys = []string{}
		}
		s.lock.Unlock()
	} else {
		keys = []string{}
	}
	return
}
func (s *store) _putValues(path string, data []string) {
	s.removeIndexes(path)
	s.data[path] = data
	if s.buildIndex {
		for _, i := range data {
			s.index[i] = append(s.index[i], path)
		}
	}
	s.changed = true
}

func (s *store) PutValues(path string, data []string) {
	s.lock.Lock()
	s._putValues(path, data)
	s.lock.Unlock()
}

func (s *store) GetValues(path string) (values []string) {
	var ok bool
	s.lock.Lock()
	if values, ok = s.data[path]; !ok {
		values = []string{}
	}
	s.lock.Unlock()
	return
}

func (s *store) Save(path string) error {
	var buffer bytes.Buffer
	/// FIXME: need to escape string in fact or things go wrong
	s.lock.Lock()
	for k, v := range s.data {
		data := k + separator + strings.Join(v, separator) + "\n"
		// FIXME: error processing
		buffer.WriteString(data)
	}
	s.changed = false
	s.lock.Unlock()
	return ioutil.WriteFile(path, buffer.Bytes(), 0755)
}

func (s *store) Restore(path string, buildIndex bool) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(data)
	s.data = make(map[string][]string)
	s.index = make(map[string][]string)
	s.buildIndex = buildIndex
	s.lock.Lock()
	for buffer.Len() > 0 {
		line, err := buffer.ReadString('\n')
		if len(line) > 0 {
			parts := strings.Split(line, separator)
			if len(parts) > 1 {
				for i, _ := range parts {
					parts[i] = strings.Trim(parts[i], "\n")
				}
				s._putValues(parts[0], parts[1:])
			} else {
				logrus.Error("wrong line: %s, skip\n", line)
			}
		}
		// just defense in depth.
		if err != nil {
			break
		}
	}
	s.changed = false
	s.lock.Unlock()
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (s *store) Changed() bool {
	return s.changed
}
