# first build the image
(cd ..; DOCKER_BUILDKIT=1 docker build -f sui/Dockerfile.base -t sui .)
# tag the image with the appropriate version
docker tag sui:latest ghcr.io/wormhole-foundation/sui:0.10.0
# push to ghcr
docker push ghcr.io/wormhole-foundation/sui:0.10.0
