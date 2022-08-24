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
import { loadPlugins } from "./loadPlugins";

setDefaultWasm("node");

// instantiate common environment
const commonEnv = getCommonEnvironment();
const logger = getLogger();

async function main() {
  const plugins = await loadPlugins(commonEnv);
  if (process.env.MODE === "listener") {
    // init listener harness
    const promHelper = new PromHelper(
      "plugin_relayer",
      commonEnv.promPort,
      PromMode.Listen
    );
    redisHelper.init(promHelper);
  } else if (process.env.MODE === "executor") {
    // init executor harness
    const promHelper = new PromHelper(
      "plugin_relayer",
      commonEnv.promPort,
      PromMode.Execute
    );
    redisHelper.init(promHelper);
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
