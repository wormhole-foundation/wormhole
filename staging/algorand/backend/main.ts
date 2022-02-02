/* eslint-disable func-call-spacing */
/* eslint-disable no-unused-vars */
/**
 * Pricecaster Service.
 *
 * Main program file.
 *
 * Copyright 2022 Wormhole Project Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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
  console.log('Pricecaster Service Backend  -- (c) 2022 Wormhole Project Contributors\n')
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
