# first build the image
(cd ..; DOCKER_BUILDKIT=1 docker build --progress plain -f sui/Dockerfile.base -t sui .)
# tag the image with the appropriate version
docker tag sui:latest ghcr.io/wormhole-foundation/sui:0.11.0-rh
# push to ghcr
docker push ghcr.io/wormhole-foundation/sui:0.11.0-rh

echo remember to update both Dockerfile and Dockerfile.export
