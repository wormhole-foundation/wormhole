module.exports = {
  pollInterval: 4,
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
    // token: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    // api: 'http:localhost//127.0.0.1',
    // port: '4001',
    token: '',
    api: 'https://api.testnet.algoexplorer.io',
    port: '',
    dumpFailedTx: true,
    dumpFailedTxDirectory: './dump'
  },
  apps: {
    vaaVerifyProgramBinFile: 'bin/vaa-verify.bin',
    vaaProcessorAppId: 74563743,
    priceKeeperV2AppId: 74563760,
    vaaVerifyProgramHash: 'NDCOXBIMDEVFYXL6HXZSA5MXFRXC276R4YRJEDLYOHK7WJC2VYKSXJCUHM',
    ownerAddress: 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU',
    ownerKeyFile: './keys/owner.key'
  },
  pyth: {
    chainId: 1,
    emitterAddress: 'f346195ac02f37d60d4db8ffa6ef74cb1be3550047543a4a9ee9acf4d78697b0'
  },
  wormhole: {
    spyServiceHost: 'natasha.randlabs.io:7073'
  },
  symbols: {
    sourceNetwork: 'devnet'
  }
}
