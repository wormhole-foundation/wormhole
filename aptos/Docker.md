# build the image and tag it appropriately

<!-- cspell:disable -->

```bash
cd .. && docker buildx build --platform linux/amd64,linux/arm64 -f aptos/Dockerfile.base -t ghcr.io/wormhole-foundation/aptos:3.1.0 .
```

<!-- cspell:enable -->

# push to ghcr

```bash
docker push ghcr.io/wormhole-foundation/aptos:3.1.0
```
