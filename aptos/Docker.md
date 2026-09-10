# build the image and tag it appropriately

<!-- cspell:disable -->

```bash
docker buildx build --platform linux/amd64,linux/arm64 -f Dockerfile.base -t ghcr.io/wormhole-foundation/aptos:4.5.0 .
```

<!-- cspell:enable -->

# push to ghcr

```bash
docker push ghcr.io/wormhole-foundation/aptos:4.5.0
```
