/*
1 Load config files onto process environment
2 Init Logging
3 Create The Common Environment
4 Init Plugins
5 Branch, based on Listen or Execute

Listen Path
-Init Listener

Execute
-Init Executor
*/

require("./helpers/loadConfig");

import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getCommonEnvironment } from "./configureEnv";
import { getLogger } from "./helpers/logHelper";
import { PromHelper, PromMode } from "./helpers/promHelpers";
import * as redisHelper from "./helpers/redisHelper";
import { IPlugin, Plugin, Listener } from "./pluginInterface";

setDefaultWasm("node");

// instantiate common environment
const commonEnv = getCommonEnvironment();
const logger = getLogger();
async function loadPlugins(promHelper: PromHelper): Promise<IPlugin[]> {
  /*
  1. read plugin URIs from common config
  For Each
    a. dynamically load plugin
    b. look for plugin overrides in common config
    c. construct plugin 
  */
  const uris = commonEnv.pluginUris
  for (const uri of uris) {
    const pluginClass = require(uri) as any;
    const plugin = new pluginClass(promHelper, commonEnv, commonEnv.plugins[uri]) as Plugin;
    const metrics = plugin.defineMetrics()
    promHelper.
  }
  return [];
}

async function main() {
  if (process.env.MODE === "listener") {
    // init listener harness
    const promHelper = new PromHelper(
      "plugin_relayer",
      commonEnv.promPort,
      PromMode.Listen
    );
    redisHelper.init(promHelper);
    const plugins = await loadPlugins(promHelper);


  } else if (process.env.MODE === "executor") {
    // init executor harness
    const promHelper = new PromHelper(
      "plugin_relayer",
      commonEnv.promPort,
      PromMode.Execute
    );
    redisHelper.init(promHelper);
    const plugins = await loadPlugins(promHelper);
  } else {
    throw new Error(
      "Expected MODE env var to be listener or executor, instead got: " +
        process.env.MODE
    );
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
