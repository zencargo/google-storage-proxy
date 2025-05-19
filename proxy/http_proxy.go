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
	bucketHandler          *storage.BucketHandle
	defaultGCSObjectPrefix string
	bucketName             string
	stripPathPrefix        string
}

func NewStorageProxy(bucketHandler *storage.BucketHandle, defaultGCSObjectPrefix string, bucketName string, stripPathPrefix string) *StorageProxy {
	return &StorageProxy{
		bucketHandler:          bucketHandler,
		defaultGCSObjectPrefix: defaultGCSObjectPrefix,
		bucketName:             bucketName,
		stripPathPrefix:        stripPathPrefix,
	}
}

func (proxy StorageProxy) objectName(nameAfterStripping string) string {
	if strings.HasPrefix(nameAfterStripping, proxy.bucketName+"/") {
		return strings.TrimPrefix(nameAfterStripping, proxy.bucketName+"/")
	}
	return proxy.defaultGCSObjectPrefix + nameAfterStripping
}

func (proxy StorageProxy) Serve(address string, port int64) error {
	listenAddr := fmt.Sprintf("%s:%d", address, port)
	http.HandleFunc("/", proxy.handler)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("gcs-proxy: failed to listen on %s: %v", listenAddr, err)
		return fmt.Errorf("gcs-proxy: failed to listen on %s: %w", listenAddr, err)
	}

	log.Printf("Zencargo GCS Proxy listening on %s", listener.Addr().String())
	log.Printf("gcs-proxy: All requests to path / will be handled by proxy.handler")

	serverErr := http.Serve(listener, nil)
	if serverErr != nil && serverErr != http.ErrServerClosed {
		log.Printf("gcs-proxy: HTTP server error on %s: %v", listener.Addr().String(), serverErr)
		return fmt.Errorf("gcs-proxy: http server error: %w", serverErr)
	}
	log.Printf("Zencargo GCS Proxy on %s shut down.", listener.Addr().String())
	return nil
}

func (proxy StorageProxy) handler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path

	if proxy.stripPathPrefix != "" {
		if strings.HasPrefix(key, proxy.stripPathPrefix) {
			key = strings.TrimPrefix(key, proxy.stripPathPrefix)
		} else {
			log.Printf("gcs-proxy: Warning - request path %q did not have expected strip-prefix %q. Will attempt to serve as is relative to root.", r.URL.Path, proxy.stripPathPrefix)
		}
	}

	if key == "" {
		key = "index.html"
	}

	if len(key) > 0 && key[0] == '/' {
		key = key[1:]
	}

	if key == "" {
		key = "index.html"
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
	objectNameInBucket := proxy.objectName(name)

	object := proxy.bucketHandler.Object(objectNameInBucket)
	reader, err := object.NewReader(context.Background())
	if err != nil {
		log.Printf("Error creating reader for GCS object %q (it may not exist or there are permission issues): %v", objectNameInBucket, err) // Keep error log
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer reader.Close()

	bufferedReader := bufio.NewReader(reader)
	_, err = bufferedReader.WriteTo(w)
	if err != nil {
		log.Printf("Failed to write GCS object %q to HTTP response: %v", objectNameInBucket, err)
	} else {
		log.Printf("Successfully served GCS object: %q", objectNameInBucket)
	}
}

func (proxy StorageProxy) checkBlobExists(w http.ResponseWriter, name string) {
	objectNameInBucket := proxy.objectName(name)

	object := proxy.bucketHandler.Object(objectNameInBucket)
	attrs, err := object.Attrs(context.Background())
	if err != nil {
		log.Printf("Error fetching attributes for GCS object %q (or object not found): %v", objectNameInBucket, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if attrs == nil {
		log.Printf("Attributes are nil for GCS object %q (unexpected)", objectNameInBucket)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (proxy StorageProxy) uploadBlob(w http.ResponseWriter, r *http.Request, name string) {
	objectNameInBucket := proxy.objectName(name)
	object := proxy.bucketHandler.Object(objectNameInBucket)
	writer := object.NewWriter(context.Background())

	var writeSuccessful bool = false
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("Failed to close GCS object writer for %q: %v", objectNameInBucket, err)
			if writeSuccessful {
			}
		} else if writeSuccessful {
			log.Printf("Successfully uploaded and finalized GCS object: %q", objectNameInBucket)
		}
	}()

	bufferedReader := bufio.NewReader(r.Body)
	bufferedWriter := bufio.NewWriter(writer)

	_, err := bufferedWriter.ReadFrom(bufferedReader)
	if err != nil {
		log.Printf("Failed during ReadFrom (request body to GCS writer) for blob %q: %v", objectNameInBucket, err)
		uploadBlobFailedResponse(w, err)
		return
	}

	if err := bufferedWriter.Flush(); err != nil {
		log.Printf("Failed to flush writer for blob %q: %v", objectNameInBucket, err)
		uploadBlobFailedResponse(w, err)
		return
	}

	writeSuccessful = true
	w.WriteHeader(http.StatusCreated)
}

func uploadBlobFailedResponse(w http.ResponseWriter, err error) {
	log.Printf("Upload failed for object: %v", err)
	http.Error(w, fmt.Sprintf("Blob upload failed: %v", err), http.StatusBadRequest)
}
