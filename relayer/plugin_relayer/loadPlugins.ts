import { PluginFactory, Plugin, CommonEnvironment } from "plugin_interface";
import { getLogger } from "./helpers/logHelper";

/*
  1. read plugin URIs from common config
  For Each
    a. dynamically load plugin
    b. look for plugin overrides in common config
    c. construct plugin 
 */
export async function loadPlugins(
  commonEnv: CommonEnvironment
): Promise<Plugin[]> {
  const logger = getLogger();
  logger.info("Loading plugins...");
  const plugins = await Promise.all(
    commonEnv.plugins.map(({ uri, overrides }) =>
      loadPlugin(uri, overrides, commonEnv)
    )
  );
  logger.info(`Loaded ${plugins.length} plugins`);
  return plugins
}

export async function loadPlugin(
  uri: string,
  overrides: { [key: string]: any } | undefined,
  commonEnv: CommonEnvironment
): Promise<Plugin> {
  const module = (await import(uri)) as PluginFactory;
  return module.create(commonEnv, overrides);
}

/* uncomment and run with ts-node loadPlugins.ts to test separately */

// loadPlugins({
//   plugins: [{ uri: "dummy_plugin", overrides: {key: "val"} }],
//   logLevel: "",
//   promPort: 0,
//   redisHost: "",
//   redisPort: 0,
//   envType: "",
// }).then((e) => {
//   console.error(e);
//   process.exit(1);
// });
