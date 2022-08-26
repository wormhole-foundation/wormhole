# first build the image

DOCKER_BUILDKIT=1 docker build -f Dockerfile.base -t near .

# tag the image with the appropriate version

docker tag near:latest ghcr.io/wormhole-foundation/near:0.1

# push to ghcr

docker push ghcr.io/wormhole-foundation/near:0.1
