#!/usr/bin/env bash
npm run cleanup
if [! docker info > /dev/null ] ; then
  echo "This script uses docker, and it isn't running - please start docker and try again!"
  exit 1
fi

# Check if wormhole/ repo exists.
# If it doens't then clone and build guardiand
if [ ! -d "./wormhole" ] 
then
    git clone https://github.com/certusone/wormhole
    cd wormhole/
    DOCKER_BUILDKIT=1 docker build --target go-export -f Dockerfile.proto -o type=local,dest=node .
    DOCKER_BUILDKIT=1 docker build --target node-export -f Dockerfile.proto -o type=local,dest=. .
    cd node/
    echo "Have patience, this step takes upwards of 500 seconds!"
    if [ $(uname -m) = "arm64" ]; then
        echo "Building Guardian for linux/amd64"
        DOCKER_BUILDKIT=1 docker build --platform linux/amd64 -f Dockerfile -t guardian .
    else 
        echo "Building Guardian natively"
        DOCKER_BUILDKIT=1 docker build -f Dockerfile -t guardian .
    fi
    cd ../../
fi

# Start EVM Chain 0
npx pm2 start 'ganache -p 8545 -m "myth like bonus scare over problem client lizard pioneer submit female collect" --block-time 1' --name evm0
# Start EVM Chain 1
npx pm2 start 'ganache -p 8546 -m "myth like bonus scare over problem client lizard pioneer submit female collect" --block-time 1' --name evm1
#Install Wormhole Eth Dependencies
cd wormhole/ethereum
npm i
cp .env.test .env

npm run build

# Deploy Wormhole Contracts to EVM Chain 0
npm run migrate && npx truffle exec scripts/deploy_test_token.js && npx truffle exec scripts/register_solana_chain.js && npx truffle exec scripts/register_terra_chain.js && npx truffle exec scripts/register_bsc_chain.js && npx truffle exec scripts/register_algo_chain.js
# Deploy Wormhole Contracts to EVM Chain 1
perl -pi -e 's/CHAIN_ID=0x2/CHAIN_ID=0x4/g' .env && perl -pi -e 's/8545/8546/g' truffle-config.js 
npm run migrate && npx truffle exec scripts/deploy_test_token.js && npx truffle exec scripts/register_solana_chain.js && npx truffle exec scripts/register_terra_chain.js && npx truffle exec scripts/register_eth_chain.js && npx truffle exec scripts/register_algo_chain.js && nc -lkp 2000 0.0.0.0
perl -pi -e 's/CHAIN_ID=0x4/CHAIN_ID=0x2/g' .env && perl -pi -e 's/8546/8545/g' truffle-config.js
cd ../../

# Run Guardiand
if [ $(uname -m) = "arm64" ]; then
    docker run -d --name guardiand -p 7070:7070 -p 7071:7071 -p 7073:7073 --platform linux/amd64 --hostname guardian-0 --cap-add=IPC_LOCK --entrypoint /guardiand guardian node \
        --unsafeDevMode --guardianKey /tmp/bridge.key --publicRPC "[::]:7070" --publicWeb "[::]:7071" --adminSocket /tmp/admin.sock --dataDir /tmp/data \
        --ethRPC ws://host.docker.internal:8545 \
        --ethContract "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" \
        --bscRPC ws://host.docker.internal:8546 \
        --bscContract "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" \
        --polygonRPC ws://host.docker.internal:8545 \
        --avalancheRPC ws://host.docker.internal:8545 \
        --auroraRPC ws://host.docker.internal:8545 \
        --fantomRPC ws://host.docker.internal:8545 \
        --oasisRPC ws://host.docker.internal:8545 \
        --karuraRPC ws://host.docker.internal:8545 \
        --acalaRPC ws://host.docker.internal:8545 \
        --klaytnRPC ws://host.docker.internal:8545 \
        --celoRPC ws://host.docker.internal:8545 \
        --moonbeamRPC ws://host.docker.internal:8545 \
        --neonRPC ws://host.docker.internal:8545 \
        --terraWS ws://host.docker.internal:8545 \
        --terra2WS ws://host.docker.internal:8545 \
        --terraLCD https://host.docker.internal:1317 \
        --terra2LCD http://host.docker.internal:1317  \
        --terraContract terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5 \
        --terra2Contract terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5 \
        --solanaContract Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o \
        --solanaWS ws://host.docker.internal:8900 \
        --solanaRPC http://host.docker.internal:8899 \
        --algorandIndexerRPC ws://host.docker.internal:8545 \
        --algorandIndexerToken "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
        --algorandAlgodToken "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
        --algorandAlgodRPC https://host.docker.internal:4001 \
        --algorandAppID "4"
else 
    docker run -d --name guardiand --network host --hostname guardian-0 --cap-add=IPC_LOCK --entrypoint /guardiand guardian node \
            --unsafeDevMode --guardianKey /tmp/bridge.key --publicRPC "[::]:7070" --publicWeb "[::]:7071" --adminSocket /tmp/admin.sock --dataDir /tmp/data \
            --ethRPC ws://localhost:8545 \
            --ethContract "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" \
            --bscRPC ws://localhost:8546 \
            --bscContract "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" \
            --polygonRPC ws://localhost:8545 \
            --avalancheRPC ws://localhost:8545 \
            --auroraRPC ws://localhost:8545 \
            --fantomRPC ws://localhost:8545 \
            --oasisRPC ws://localhost:8545 \
            --karuraRPC ws://localhost:8545 \
            --acalaRPC ws://localhost:8545 \
            --klaytnRPC ws://localhost:8545 \
            --celoRPC ws://localhost:8545 \
            --moonbeamRPC ws://localhost:8545 \
            --neonRPC ws://localhost:8545 \
            --terraWS ws://localhost:8545 \
            --terra2WS ws://localhost:8545 \
            --terraLCD https://terra-terrad:1317 \
            --terra2LCD http://localhost:1317  \
            --terraContract terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5 \
            --terra2Contract terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5 \
            --solanaContract Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o \
            --solanaWS ws://localhost:8900 \
            --solanaRPC http://localhost:8899 \
            --algorandIndexerRPC ws://localhost:8545 \
            --algorandIndexerToken "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
            --algorandAlgodToken "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
            --algorandAlgodRPC https://localhost:4001 \
            --algorandAppID "4"
fi
echo "Guardiand Running! To look at logs: \"docker logs guardiand -f\""