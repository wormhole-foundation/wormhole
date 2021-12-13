module.exports = {
  algo: {
    token: '',
    api: 'https://api.testnet.algoexplorer.io',
    port: ''
  },
  pyth: {
    chainId: 1,
    emitterAddress: '3afda841c1f43dd7d546c8a581ba1f92a139f4133f9f6ab095558f6a359df5d4'
  },
  wormhole: {
    spyServiceHost: 'localhost:7073'
  },
  strategy: {
    bufferSize: 100
  },
  symbols: [
    {
      name: 'ETH/USD',
      productId: '0xc67940be40e0cc7ffaa1acb08ee3fab30955a197da1ec297ab133d4d43d86ee6',
      priceId: '0xff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace',
      publishIntervalSecs: 30,
      priceKeeperV2AppId: 32984466
    },
    {
      name: 'ALGO/USD',
      productId: '0x30fabb4e8ee48aec78799e8835c1b744d10d212c64f2671bed98d7b76a5306b0',
      priceId: '0xfa17ceaf30d19ba51112fdcc750cc83454776f47fb0112e4af07f15f4bb1ebc0',
      publishIntervalSecs: 15,
      priceKeeperV2AppId: 32984466
    },
    {
      productId: '3515b3861e8fe93e5f540ba4077c216404782b86d5e78077b3cbfd27313ab3bc',
      priceId: '0xe62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43',
      name: 'BTC/USD',
      publishIntervalSecs: 25,
      priceKeeperV2AppId: 32984466
    },
    {
      productId: '230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9',
      priceId: 'fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd',
      name: 'TEST',
      publishIntervalSecs: 30,
      priceKeeperV2AppId: 1
    }
  ]
}
