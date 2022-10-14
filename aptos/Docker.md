# first build the image
(cd ..; DOCKER_BUILDKIT=1 docker build -f aptos/Dockerfile.base -t aptos .)
# tag the image with the appropriate version
docker tag aptos:latest ghcr.io/wormhole-foundation/aptos:0.3.4
# push to ghcr
docker push ghcr.io/wormhole-foundation/aptos:0.3.4
