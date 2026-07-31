# build the image, tag it appropriately, and push to ghcr

<!-- cspell:disable -->

```bash
cd .. && docker buildx build --platform linux/amd64,linux/arm64 -f sui/Dockerfile.base --push -t ghcr.io/wormhole-foundation/sui:[tag] .
```