package http_cache

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
)

type StorageProxy struct {
	bucketHandler *storage.BucketHandle
	defaultPrefix string
	bucketName    string
}

func NewStorageProxy(bucketHandler *storage.BucketHandle, defaultPrefix string, bucketName string) *StorageProxy {
	return &StorageProxy{
		bucketHandler: bucketHandler,
		defaultPrefix: defaultPrefix,
		bucketName:    bucketName,
	}
}

func (proxy StorageProxy) objectName(name string) string {
	if strings.HasPrefix(name, proxy.bucketName+"/") {
		return strings.TrimPrefix(name, proxy.bucketName+"/")
	}
	return proxy.defaultPrefix + name
}

func (proxy StorageProxy) Serve(address string, port int64) error {
	http.HandleFunc("/", proxy.handler)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return err
	}

	address = listener.Addr().String()
	log.Printf("Starting http cache server %s\n", address)
	log.Printf("Zencargo GCS Proxy")
	listener.Close()
	return http.ListenAndServe(address, nil)
}

func (proxy StorageProxy) handler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	if key[0] == '/' {
		key = key[1:]
	}

	ext := filepath.Ext(key)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)

	switch r.Method {
	case "GET":
		proxy.downloadBlob(w, key)
	case "HEAD":
		proxy.checkBlobExists(w, key)
	case "POST", "PUT":
		proxy.uploadBlob(w, r, key)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (proxy StorageProxy) downloadBlob(w http.ResponseWriter, name string) {
	object := proxy.bucketHandler.Object(proxy.objectName(name))
	if object == nil {
		log.Printf("Object not found: %s", name)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	reader, err := object.NewReader(context.Background())
	if err != nil {
		log.Printf("Error reading object %s: %v", name, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer reader.Close()
	bufferedReader := bufio.NewReader(reader)
	_, err = bufferedReader.WriteTo(w)
	if err != nil {
		log.Printf("Failed to serve blob %q: %v", name, err)
	}
}

func (proxy StorageProxy) checkBlobExists(w http.ResponseWriter, name string) {
	object := proxy.bucketHandler.Object(proxy.objectName(name))
	if object == nil {
		log.Printf("Object not found: %s", name)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	attrs, err := object.Attrs(context.Background())
	if err != nil || attrs == nil {
		log.Printf("Error fetching attributes for object %s: %v", name, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (proxy StorageProxy) uploadBlob(w http.ResponseWriter, r *http.Request, name string) {
	object := proxy.bucketHandler.Object(proxy.objectName(name))

	writer := object.NewWriter(context.Background())
	defer writer.Close()

	bufferedWriter := bufio.NewWriter(writer)
	bufferedReader := bufio.NewReader(r.Body)

	_, err := bufferedWriter.ReadFrom(bufferedReader)
	if err != nil {
		uploadBlobFailedResponse(w, err)
		return
	}

	if err := bufferedWriter.Flush(); err != nil {
		uploadBlobFailedResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func uploadBlobFailedResponse(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	errorMsg := fmt.Sprintf("Blob upload failed: %v", err)
	w.Write([]byte(errorMsg))
}
