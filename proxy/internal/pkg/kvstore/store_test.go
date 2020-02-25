package kvstore

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func reset() {
	writeTimeout = 2
	simpleStoreLock.Lock()
	watchedSimpleStores = make(map[string]WatchedStore)
	simpleStoreLock.Unlock()
	done <- true
}

func checkStoreExist(t *testing.T, names ...string) {
	simpleStoreLock.Lock()
	for _, name := range names {
		assert.Contains(t, watchedSimpleStores, name, "should contain "+name)
	}
	simpleStoreLock.Unlock()
}

var (
	counter int32 = 0
)

func TestNew(t *testing.T) {
	writeTimeout = 2
	value := atomic.AddInt32(&counter, 1)
	_, err := NewStore("s1", true)
	assert.Nil(t, err, "should create s1")
	s2, err := NewStore("s2", true)
	assert.Nil(t, err, "should create s2")
	s2_other, err := NewStore("s2", true)
	assert.Equal(t, s2, s2_other, "should fetch existing repo")
	checkStoreExist(t, "s1", "s2")
	fmt.Printf("counter is %d\n", value)
}

func TestMethods(t *testing.T) {
	value := atomic.AddInt32(&counter, 1)
	s := store{
		data:       make(map[string][]string),
		index:      make(map[string][]string),
		changed:    true, // need write for init
		buildIndex: true,
	}
	s.Put("key1", "data1")
	s.Put("key2", "data2")
	s.Put("key3", "data3")
	assert.Equal(t, s.Get("key1"), "data1", "test get")
	assert.Equal(t, s.Get("key2"), "data2", "test get")
	assert.Equal(t, s.Get("key3"), "data3", "test get")
	s.Put("key1", "data2")
	assert.Equal(t, s.Get("key1"), "data2", "test overwrite")
	assert.Equal(t, s.GetKey("data1"), []string{}, "test overwrite consistency")
	s.PutValues("mkey1", []string{"data1", "data2", "data3"})
	assert.Contains(t, s.GetValues("mkey1"), "data1", "test multiput")
	assert.Contains(t, s.GetValues("mkey1"), "data2", "test multiput")
	assert.Equal(t, s.Get("mkey1"), "data1", "test get single from multi-value")
	assert.Contains(t, s.GetKey("data3"), "key3", "test getkey single")
	assert.Contains(t, s.GetKey("data2"), "key1", "test getkey multi")
	assert.Contains(t, s.GetKey("data2"), "key2", "test getkey multi")
	assert.Contains(t, s.GetKey("data2"), "mkey1", "test getkey multi")
	fmt.Printf("counter is %d\n", value)
}

func TestNoIndexMethods(t *testing.T) {
	value := atomic.AddInt32(&counter, 1)
	s := store{
		data:       make(map[string][]string),
		index:      make(map[string][]string),
		changed:    true, // need write for init
		buildIndex: false,
	}
	s.Put("key1", "data1")
	s.Put("key2", "data2")
	s.Put("key3", "data3")
	assert.Equal(t, s.Get("key1"), "data1", "test get")
	assert.Equal(t, s.Get("key2"), "data2", "test get")
	assert.Equal(t, s.Get("key3"), "data3", "test get")
	s.Put("key1", "data2")
	assert.Equal(t, s.Get("key1"), "data2", "test overwrite")
	assert.Empty(t, s.GetKey("data1"), "test no get key")
	s.PutValues("mkey1", []string{"data1", "data2", "data3"})
	assert.Contains(t, s.GetValues("mkey1"), "data1", "test multiput")
	assert.Contains(t, s.GetValues("mkey1"), "data2", "test multiput")
	assert.Equal(t, s.Get("mkey1"), "data1", "test get single from multi-value")
	assert.Empty(t, s.GetKey("data3"), "test getkey single")
	assert.Empty(t, s.GetKey("data2"), "test getkey multi")
	assert.Empty(t, s.GetKey("data2"), "test getkey multi")
	assert.Empty(t, s.GetKey("data2"), "test getkey multi")
	fmt.Printf("counter is %d\n", value)
}

func TestSaveRestore(t *testing.T) {
	value := atomic.AddInt32(&counter, 1)
	s := store{
		data:       make(map[string][]string),
		index:      make(map[string][]string),
		changed:    true, // need write for init
		buildIndex: true,
	}
	s.Put("key1", "data1")
	s.Put("key2", "data2")
	s.Put("key3", "data3")
	s.PutValues("mkey1", []string{"data1", "data2", "data3"})
	s.PutValues("mkey2", []string{"datax", "data2", "data3"})
	s.PutValues("mkey3", []string{"datay", "data1", "data2"})
	s.Save("tmp")

	expected := [6]string{
		"key1;data1\n",
		"key2;data2\n",
		"key3;data3\n",
		"mkey1;data1;data2;data3\n",
		"mkey2;datax;data2;data3\n",
		"mkey3;datay;data1;data2\n",
	}

	actual, _ := ioutil.ReadFile("tmp")
	for i := 0; i < 6; i++ {
		assert.Contains(t, string(actual), expected[i], "test file equality")
	}

	var y store
	y.Restore("tmp", true)
	assert.Equal(t, s.Get("key1"), "data1", "test recover")
	assert.Equal(t, s.Get("mkey2"), "datax", "test recover")
	assert.Equal(t, s.GetValues("mkey3"), []string{"datay", "data1", "data2"}, "test recover")

	var z store
	z.Restore("tmp", false)
	assert.Empty(t, z.GetKey("data1"), "no index then return empty")

	os.Remove("tmp")
	fmt.Printf("counter is %d\n", value)
}

func TestWatchedStores(t *testing.T) {
	s1, err := NewStore("s1", true)
	assert.Nil(t, err, "creating store")
	s2, err := NewStore("s2", true)
	assert.Nil(t, err, "creating store")

	s1.Put("key1", "data1")
	s1.Put("key2", "data2")
	s1.PutValues("mkey1", []string{"data1", "data2", "data3"})
	s2.PutValues("mkey1", []string{"data1", "data2", "data3"})
	s2.PutValues("mkey2", []string{"datax", "dataz", "data2"})
	s2.PutValues("mkey3", []string{"datay", "datau", "data3"})
	time.Sleep(time.Duration(writeTimeout+1) * time.Second)

	fmt.Printf("here\n")
	info, err := os.Stat(filepath.Join(storeDir, "s1"))
	assert.Nil(t, err, "fetching first store")
	mtime := info.ModTime()
	_, err = os.Stat(filepath.Join(storeDir, "s2"))
	assert.Nil(t, err, "fetching second store")
	s1.Put("keyx", "datax")
	fmt.Printf("here\n")
	info, err = os.Stat(filepath.Join(storeDir, "s1"))
	assert.True(t, mtime.Equal(info.ModTime()), "modify time")
	time.Sleep(time.Duration(writeTimeout+1) * time.Second)
	info, err = os.Stat(filepath.Join(storeDir, "s1"))
	assert.False(t, mtime.Equal(info.ModTime()), "modify time")

	fmt.Printf("here\n")
	reset()
	fmt.Printf("here\n")
	assert.Nil(t, RestartStore(true), "recover store")
	r1 := GetStore("s1")
	assert.NotNil(t, r1, "get recovered store")
	assert.Equal(t, r1.Get("keyx"), "datax", "get data")
	assert.Contains(t, r1.GetKey("data1"), "key1", "get key")
	assert.Contains(t, r1.GetKey("data1"), "mkey1", "get key")
	assert.NotNil(t, GetStore("s1"), "get recovered store")

}
