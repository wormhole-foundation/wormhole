FROM eth-node

USER node

WORKDIR /home/node/app

# Clone queries-staking repo from git
ARG QUERIES_STAKING_URL
ARG QUERIES_STAKING_TAG
RUN git clone -b ${QUERIES_STAKING_TAG} --single-branch ${QUERIES_STAKING_URL} queries-staking

RUN cd queries-staking && forge install

# Overlay our custom scripts
COPY --chown=node:node wormhole/devnet/staking-contracts/deploy_all_pools.sh queries-staking/
# COPY --chown=node:node wormhole/devnet/staking-contracts/distribute_test_tokens.sh queries-staking/
COPY --chown=node:node wormhole/devnet/staking-contracts/script/CreateStakingPool.s.sol queries-staking/script/

WORKDIR /home/node/app/queries-staking

# Make deploy scripts executable
RUN chmod +x *.sh
