#!/bin/bash

set -meuo pipefail

TLS_HOSTNAME="Guardian"
TLS_PUBLIC_IP="127.0.0.1"
TLS_PORT="3003"
PEER_SERVER_URL="http://127.0.0.1:3000"

GUARDIAN_PRIVATE_KEYS=(
  "c67c82a42364074e4a3ffec944a96d758cec08da09275ada471fe82c95159af9"
  "22b517009ccd7ede90cebf16ab630868989172c72ebd70ffc6d04cb783eeba61"
  "82272a56366878aa05e5cbcd5cd1d195d14ca26064be20cdb7aacebcf2c86a20"
  "6e24f50d0c1bae02d47e0ef5ae26cc5e3ee7d98660b508736fa4a7d64fe3c625"
  "c81a9b27aeef5a963959d5e424a0686a981918a009cacef18ded6e6496c9a808"
  "439bd6108f1590b56ebdea0706c2dc308240c4a47d94f42d16f92db81f6222b9"
  "a2218874818e7da07f632a9c9bbfcd96ed478b6a4a6a9829bff2eb5f8ed719a8"
  "d73ba7c666d41f554e8beaf96d5e2ae6590aefc7b5ef7008dc0b49b6328966a0"
  "7b015af9ed3b47ff5a1a71410bbda0aaf40111858925784658f19acf4c085f2e"
  "fe156e000de4ae76fd51fb41331bbab47a10a6f0d8ea96859cd5f66e2ec9b6a9"
  "ebbe5853f748ac603c984ed162ee44d2e898ca4552f3b153f59a665681ab2c00"
  "155f68801d5a51f216b26a3e0f5fb41a1f0525a61cddb6b0ab2b4c19213ebc55"
  "2c45b839f24d1cb4ea25c0079e62f930f91499c77d13b43074b3c3810718cd8d"
  "61bad7e65e328ae170db4326021c7585baadae785350e12f1f2bdb7e73c10b28"
  "9f8efce656540a4ddd16a323d7a3e2e9355dc13c7afa1f6197ea72cfc1375517"
  "ef60c2cd0f8ae627298a1dd237670198aa1dabb3cbcf2a532fd1cadc698788e3"
  "83b81eb33a7fcb0694e212de9e4fb0db2579e8c4941a43adae1ebd8f43559aa9"
  "24b2e1ac892f26e3741cd99b29ceb4c9f6acadb1af74942e83f23f8d37dcb6d6"
  "759540959b56070ee6effb4793af059d49897d9b54edaf78f0f06f5b03ec6c2c"
)

createGuardianPrivateKey() {
    echo 0a20${GUARDIAN_PRIVATE_KEYS[$1]} | \
        xxd -r -p | gpg --enarmor | \
        awk 'BEGIN {print "-----BEGIN WORMHOLE GUARDIAN PRIVATE KEY-----"}
             NR>2 {print last}
             {last=$0}
             END {print "-----END WORMHOLE GUARDIAN PRIVATE KEY-----"}'
}

export DOCKER_BUILDKIT=1
# Build the docker cache first. It will throw an error but it will save time
docker build --network="host" -f ../../docker/Dockerfile --progress=plain . || true

for i in "${!GUARDIAN_PRIVATE_KEYS[@]}"
do
    docker build --network="host" -f ../../docker/Dockerfile \
        --secret id=guardian_pk,src=<(createGuardianPrivateKey $i) \
        --build-arg TLS_HOSTNAME=${TLS_HOSTNAME}$i \
        --build-arg TLS_PUBLIC_IP=${TLS_PUBLIC_IP} \
        --build-arg TLS_PORT=${TLS_PORT} \
        --build-arg PEER_SERVER_URL=${PEER_SERVER_URL} \
        --progress=plain --output=out/$i/ . &
done

