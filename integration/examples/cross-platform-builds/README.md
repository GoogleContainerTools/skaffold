### Example: Getting started with a simple go app built for specified platform

Run:
```
skaffold build --default-repo=gcr.io/<your-repo> --platform=linux/arm64 --cache-artifacts=false
```

This will build for the `linux/arm64` platform and push the image. You can test it by running:

```
docker run --platform linux/arm64 --rm -it gcr.io/<your-repo>/image:tag
```