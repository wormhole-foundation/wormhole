# first build the image
DOCKER_BUILDKIT=1 docker build -f Dockerfile.base -t aptos .
# tag the image with the appropriate version
docker tag aptos:latest ghcr.io/certusone/aptos:0.1
# push to ghcr
docker push ghcr.io/certusone/aptos:0.1
