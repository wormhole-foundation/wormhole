# first build the image

cd ..; DOCKER_BUILDKIT=1 docker build --no-cache --progress plain -f sui/Dockerfile.base -t sui .

# tag the image with the appropriate version

docker tag sui:latest ghcr.io/wormhole-foundation/sui:1.19.1-mainnet

# push to ghcr

docker push ghcr.io/wormhole-foundation/sui:1.19.1-mainnet

echo remember to update both Dockerfile and Dockerfile.export
