const path = require('path')

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
    emitterAddress: 'f346195ac02f37d60d4db8ffa6ef74cb1be3550047543a4a9ee9acf4d78697b0'
  },
  wormhole: {
    spyServiceHost: 'natasha.randlabs.io:7073'
  },
  symbols: {
    sourceNetwork: 'devnet'
  }
}
