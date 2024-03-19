# devnet wormchain client config

This folder contains config for running `wormchaind` against the devnet (Tilt) wormchain instance.

### examples

transfer `utest` from the account used by `wormchain-0` to the account used by `wormchain-1`, to smoke-test wormchain - make sure we can connect to the RPC port, the accounts exist, and wormchain is producing blocks.
<!-- cspell:disable-next-line -->
    ./build/wormchaind --home build tx bank send wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq  wormhole1wqwywkce50mg6077huy4j9y8lt80943ks5udzr  1utest --from wormchain-0  --yes --broadcast-mode block --keyring-backend test
