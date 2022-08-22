/*
1. Grab Logger & Common Env
2. Instantiate Listener Env
3. Instantiate Redis Connection
4. Optionally Instantiate Spy Connection
5. Optionally Instantiate REST Connection
6.  
*/

import { getCommonEnvironment, getListenerEnvironment } from './configureEnv'
import { getLogger, getScopedLogger } from './helpers/logHelper'
import * as redisHelper from './helpers/redisHelper'
import {IPlugin} from './pluginInterface'

const logger = getScopedLogger(["listenerHarness"], getLogger())
const commonEnv = getCommonEnvironment()

export async function run(plugins: IPlugin[]) {
  const listnerEnv = getListenerEnvironment()
}