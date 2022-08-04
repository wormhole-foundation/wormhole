# first build the image
DOCKER_BUILDKIT=1 docker build -f Dockerfile.base -t near .
# tag the image with the appropriate version
docker tag near:latest ghcr.io/certusone/near:0.1
# push to ghcr
docker push ghcr.io/certusone/near:0.1
