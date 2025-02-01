# build the image and tag it appropriately

<!-- cspell:disable -->

```bash
docker buildx build --platform linux/amd64,linux/arm64 -f Dockerfile.base -t ghcr.io/wormhole-foundation/near:2.4.0 .
```

<!-- cspell:enable -->

# push to ghcr

```bash
docker push ghcr.io/wormhole-foundation/near:2.4.0
```

# note: if unable to build amd64

M-series Macs may encounter an `illegal instruction` error when building the amd64 version of this image. Perform the arm64 build from a Mac and the amd64 build from another machine with native support. Then use a command like the following to create the multi-platform index from those manifests.

<!-- cspell:disable -->

```bash
docker buildx imagetools create -t ghcr.io/wormhole-foundation/near:2.4.0 ghcr.io/wormhole-foundation/near:2.4.0@sha256:<arm64-sha> ghcr.io/wormhole-foundation/near:2.4.0@sha256:<amd64-sha>
```

<!-- cspell:enable -->
