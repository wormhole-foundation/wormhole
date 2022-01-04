const path = require('path')

module.exports = {
  log: {
    appName: 'pricecaster-v2',
    disableConsoleLog: false,
    fileLog: {
      dir: './log',
      daysTokeep: 7
    },
    // sysLog: {
    //   host: '127.0.0.1',
    //   port: 514,
    //   transport: 'udp',
    //   protocol: 'bsd',
    //   sendInfoNotifications: false
    // },
    debugLevel: 1
  },
  algo: {
    token: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    api: 'http://127.0.0.1',
    port: '4001',
    dumpFailedTx: true,
    dumpFailedTxDirectory: './dump'
  },
  apps: {
    vaaVerifyProgramBinFile: 'bin/vaa-verify.bin',
    vaaProcessorAppId: 622608992,
    priceKeeperV2AppId: 622609307,
    vaaVerifyProgramHash: 'ISTS5S7JLD5FBLM27NW7IWMQC4XPUOGGPFHOCEOL22Q557BIDOXHENLI6Y',
    ownerAddress: 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU',
    ownerKeyFile: './keys/owner.key'
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
      productId: 'c67940be40e0cc7ffaa1acb08ee3fab30955a197da1ec297ab133d4d43d86ee6',
      priceId: 'ff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace',
      publishIntervalSecs: 30
    },
    {
      name: 'ALGO/USD',
      productId: '30fabb4e8ee48aec78799e8835c1b744d10d212c64f2671bed98d7b76a5306b0',
      priceId: 'fa17ceaf30d19ba51112fdcc750cc83454776f47fb0112e4af07f15f4bb1ebc0',
      publishIntervalSecs: 15
    },
    {
      productId: '3515b3861e8fe93e5f540ba4077c216404782b86d5e78077b3cbfd27313ab3bc',
      priceId: 'e62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43',
      name: 'BTC/USD',
      publishIntervalSecs: 25
    },
    {
      productId: '230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9',
      priceId: 'fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd',
      name: 'TEST',
      publishIntervalSecs: 20
    }
  ]
}
