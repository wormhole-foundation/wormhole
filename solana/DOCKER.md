# Docker images

To speed up builds and ensure that upstream dependencies remain available, we
publish prebuilt docker images to https://github.com/orgs/certusone/packages.

The base images have names ending in `*.base`, such as `Dockerfile.base` and
`Dockerfile.wasm.base`. To push a new image:

```sh
# first build the image
DOCKER_BUILDKIT=1 docker build -f Dockerfile.wasm.base -t wasm-pack .
# tag the image with the appropriate version
docker tag wasm-pack:latest ghcr.io/certusone/wasm-pack:0.9.1
# push to ghcr
docker push ghcr.io/certusone/wasm-pack:0.9.1
```

Finally, modify the reference in `Dockerfile.wasm` (make sure to update the sha256 hash too).
