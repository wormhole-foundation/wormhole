# first build the image

<!-- cspell:disable-next-line -->
(cd ..; DOCKER_BUILDKIT=1 docker buildx build --platform linux/amd64 -f aptos/Dockerfile.base -t aptos .)

# tag the image with the appropriate version

docker tag aptos:latest ghcr.io/wormhole-foundation/aptos:3.1.0

# push to ghcr

docker push ghcr.io/wormhole-foundation/aptos:3.1.0
