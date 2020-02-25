package kvstore

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	jhttp "github.com/jerryz920/utils/goutils/http"
	"github.com/sirupsen/logrus"
)

type CreateImageRequest struct {
	Image string
	Files []string
}

type ImageContent struct {
	Files []string
}

type IndexedImage struct {
	Image []string
}

type ImageHasher struct {
	store Store
}

func reply(v interface{}, w http.ResponseWriter) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logrus.Errorf("error encoding json in content fetching: %v", err)
	}
}

func parseBody(v interface{}, r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}

func (h *ImageHasher) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if image, ok := vars["image"]; ok {
		response := ImageContent{h.store.GetValues(image)}
		reply(&response, w)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("must provide image"))
	}
}

func (h *ImageHasher) GetIndex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if hash, ok := vars["hash"]; ok {
		response := IndexedImage{h.store.GetKey(hash)}
		reply(&response, w)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("must provide hash"))
	}

}

func (h *ImageHasher) Put(w http.ResponseWriter, r *http.Request) {
	var image CreateImageRequest
	if err := parseBody(&image, r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logrus.Errorf("error parsing body: %s", err)
		w.Write([]byte(err.Error()))
	}
	h.store.PutValues(image.Image, image.Files)
}

func NewKvStore(rootFunc http.HandlerFunc) *jhttp.APIServer {
	store, err := NewStore("kvstore", true)
	if err != nil {
		os.Exit(1)
	}
	hasher := ImageHasher{store}
	server := jhttp.NewAPIServer(rootFunc)
	server.AddRoute("/get_content/{image}", hasher.Get, "Get content of an image")
	server.AddRoute("/get_index/{hash}", hasher.GetIndex, "Index image by contents")
	server.AddRoute("/upload_image", hasher.Put, "Create new images")
	return server
}
