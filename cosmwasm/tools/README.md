## start wormchain

    cd ../wormchain && make all

    make clean && make run

## deploy core and token bridge

    npm run deploy-wormchain

this only needs to be run once after standing up wormchain.

## (try to) instantiate wormchain accounting

    npm run deploy-accounting

you can leave wormchain running and run this over and over (until it succeeds).

### debug deploy scripts

you can also run either of these with VSCode and set breakpoints and whatnot.
check `.vscode/launch.json` at the root of the repo.
