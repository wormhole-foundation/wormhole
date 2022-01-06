/* eslint-disable func-call-spacing */
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
import { WormholeClientEngine } from './engine/WormholeEngine'
import * as Logger from '@randlabs/js-logger'
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
    await Logger.initialize(settings.log)
  } catch (e: any) {
    console.error('Cannot initialize configuration: ' + e.toString())
    exit(1)
  }

  const engine = new WormholeClientEngine(settings)
  await engine.start()
})()
