## How to update image and push to dockerHub

This repo is a fork from https://github.com/cirruslabs/google-storage-proxy.

As the project is not really maintained we decided to make changes to the code in our fork and push them to our dockerHub.

- First, create a branch with updated changes.

- run `docker login` in your terminal

- Check the docker hub repository to make sure you don't build with a tag name that already exits.

- `docker build -t zencargo/google-storage-proxy:tagname .`

- Test deploying image locally in docker-desktop.

- If you're happy with the changes you can push to dockerhub.

- `docker buildx build --platform linux/amd64 -t zencargo/google-storage-proxy:tagname --push .`

---
[![Build Status](https://api.cirrus-ci.com/github/cirruslabs/google-storage-proxy.svg)](https://cirrus-ci.com/github/cirruslabs/google-storage-proxy)

HTTP proxy with REST API to interact with Google Cloud Storage Buckets

Simply allows using `HEAD`, `GET` or `PUT` requests to check blob's availability, as well as downloading or uploading
blobs to a specified GCS bucket.

Prebuilt Docker image is available on Docker Hub:

```bash
docker pull cirrusci/google-storage-proxy:latest
```

# Arguments

* `port` - optional port to run the proxy on. By default, `8080` is used.
* `bucket` - GCS bucket name to store artifacts in. You can configure [Lifecycle Management](https://cloud.google.com/storage/docs/lifecycle)
   for this bucket separately using `gcloud` or UI.
* `prefix` - optional prefix for all objects. For example, use `--prefix=foo/` to work under `foo` directory in `bucket`.
