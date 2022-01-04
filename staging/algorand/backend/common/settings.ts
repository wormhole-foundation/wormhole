/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { Options } from '@randlabs/js-logger'
import { Symbol } from './basetypes'

export interface IAppSettings extends Record<string, unknown> {
  log: Options,
  algo: {
    token: string,
    api: string,
    port: string,
    dumpFailedTx: boolean,
    dumpFailedTxDirectory?: string
  },
  apps: {
    priceKeeperV2AppId: number,
    ownerAddress: string,
    ownerKeyFile: string,
    vaaVerifyProgramBinFile: string,
    vaaVerifyProgramHash: string,
    vaaProcessorAppId: number,
  },
  pyth: {
    chainId: number,
    emitterAddress: string,
  },
  debug?: {
    logAllVaa?: boolean,
  }
  wormhole: {
    spyServiceHost: string
  },
  strategy: {
    bufferSize: number
  },
  symbols: Symbol[]
}
