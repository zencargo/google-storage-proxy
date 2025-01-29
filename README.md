## How to update image and push to Google Artifact Repository

This repo is a fork from https://github.com/cirruslabs/google-storage-proxy

The original repo is not really maintained so we decided to make changes to the code in our fork and push them to our image repository.

- First, create a branch with updated changes.

- run `docker login` in your terminal

- Check the Artifact repository to make sure you don't build/push with a tag name that already exits.
- https://console.cloud.google.com/artifacts/docker/prj-zen-c-artifact-reg-5bhv/europe-west4/google-storage-proxy/google-storage-proxy?inv=1&invt=Abn-Ig&walkthrough_id=iam--quickstart&project=prj-zen-c-artifact-reg-5bhv

- `docker buildx build --platform linux/amd64 -t zencargo/google-storage-proxy:tagname .`

- Test deploying image locally using docker-desktop.

- If you're happy with the changes you can push to Artifact Registry.

```console
$ docker tag zencargo/google-storage-proxy:tagname \
   europe-west4-docker.pkg.dev/prj-zen-c-artifact-reg-5bhv/google-storage-proxy/google-storage-proxy:tagname

$ docker push europe-west4-docker.pkg.dev/prj-zen-c-artifact-reg-5bhv/google-storage-proxy/google-storage-proxy:tagname
```

---
HTTP proxy with REST API to interact with Google Cloud Storage Buckets

Simply allows using `HEAD`, `GET` or `PUT` requests to check blob's availability, as well as downloading or uploading
blobs to a specified GCS bucket.

Prebuilt Docker image is available in Artifact Repository:

```bash
docker pull europe-west4-docker.pkg.dev/prj-zen-c-artifact-reg-5bhv/google-storage-proxy/google-storage-proxy:v3
```

# Arguments

* `port` - optional port to run the proxy on. By default, `8080` is used.
* `bucket` - GCS bucket name to store artifacts in. You can configure [Lifecycle Management](https://cloud.google.com/storage/docs/lifecycle)
   for this bucket separately using `gcloud` or UI.
* `prefix` - optional prefix for all objects. For example, use `--prefix=foo/` to work under `foo` directory in `bucket`.
