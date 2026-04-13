# build the image and tag it appropriately

<!-- cspell:disable -->

```bash
cd .. && docker buildx build --platform linux/amd64,linux/arm64 -f sui/Dockerfile.base -t ghcr.io/wormhole-foundation/sui:1.63.2-testnet .
```

<!-- cspell:enable -->

# push to ghcr

```bash
docker push ghcr.io/wormhole-foundation/sui:1.63.2-testnet
```
