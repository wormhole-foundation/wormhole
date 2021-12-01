/* eslint-disable no-unused-vars */
/**
 * Pricecaster Service.
 *
 * Main program file.
 *
 * (c) 2021 Randlabs, Inc.
 */

import * as Config from '@randlabs/js-config-reader'
import { IAppSettings } from './common/settings'
import { exit } from 'process'
import { PriceKeeperEngine } from './engine/PriceKeeperEngine'
import { WormholeClientEngine } from './engine/WormholeEngine'
const charm = require('charm')();

(async () => {
  charm.pipe(process.stdout)
  charm.reset()
  charm.foreground('cyan').display('bright')
  console.log('Pricecaster Service Backend  -- (c) 2021 Randlabs, Inc.\n')
  charm.foreground('white')

  let settings: IAppSettings
  try {
    await Config.initialize<IAppSettings>({ envVar: 'PRICECASTER_SETTINGS' })
    settings = Config.get<IAppSettings>()
  } catch (e: any) {
    console.error('Cannot initialize configuration: ' + e.toString())
    exit(1)
  }

  let engine
  switch (settings.mode) {
    case 'pkeeper':
      engine = new PriceKeeperEngine(settings)
      break

    case 'wormhole-client':
      engine = new WormholeClientEngine(settings)
      break

    default:
      console.error('Invalid specified mode in settings')
      exit(2)
  }

  engine.start()
})()
