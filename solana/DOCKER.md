# Docker images

To speed up builds and ensure that upstream dependencies remain available, we
publish prebuilt docker images to https://github.com/orgs/wormhole-foundation/packages.

The base images have names ending in `*.base`, such as `Dockerfile.base`.
To push a new image:

```sh
# first build the image
DOCKER_BUILDKIT=1 docker build -f Dockerfile.base -t solana .
# tag the image with the appropriate version
docker tag solana:latest ghcr.io/wormhole-foundation/solana:1.10.31
# push to ghcr
docker push ghcr.io/wormhole-foundation/solana:1.10.31
```

Finally, modify the reference in `Dockerfile` (make sure to update the sha256 hash too).
