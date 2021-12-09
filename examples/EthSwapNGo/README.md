## Eth Swap & Go Example Program

This is a non-production example program.

The demonstrates a simple React program which can swap native EVM currencies, using only the wormhole bridge & a relayer.

It assumes that a fresh devnet is running in conjunction with the restRelayer example project.

### In order to run this project, you must first run npm ci in the following directories:

> /examples/core
>
> /examples/rest-relayer
>
> /examples/EthSwapNGo/swapPool
>
> /examples/EthSwapNGo/react

### Once you've installed all the dependencies, you can create the swap pools:

> cd /examples/EthSwapNGo/swapPool
>
> npm run deployAndSeed

### Then start the rest-relayer service:

> cd /examples/rest-relayer
>
> npm run start

### And lastly, start the react app:

> cd /examples/EthSwapNGo/react
>
> npm run start

```

```
