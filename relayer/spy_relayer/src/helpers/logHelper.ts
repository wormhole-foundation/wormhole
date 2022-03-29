import winston = require("winston");
import { getCommonEnvironment } from "../configureEnv";

//Be careful not to access this before having called init logger, or it will be undefined
let logger: winston.Logger | undefined;

export function getLogger(): winston.Logger {
  if (logger) {
    return logger;
  } else {
    logger = initLogger();
    return logger;
  }
}

export interface ScopedLogger extends winston.Logger {
  scope?: string[];
}

// Child loggers can't override defaultMeta, they add their own defaultRequestMetadata
// ...which is stored in a closure we can't read, so we extend it ourselves :)
// https://github.com/winstonjs/winston/blob/a320b0cf7f3c550a354ce4264d7634ebc60b0a67/lib/winston/logger.js#L45
export function getScopedLogger(
  labels: string[],
  parentLogger?: ScopedLogger
): ScopedLogger {
  const scope = [...(parentLogger?.scope || []), ...labels];
  const logger = parentLogger || getLogger();
  const child: ScopedLogger = logger.child({
    labels: scope,
  });
  child.scope = scope;
  return child;
}

function initLogger(): winston.Logger {
  const loggingEnv = getCommonEnvironment();

  let useConsole = true;
  let logFileName;
  if (loggingEnv.logDir) {
    useConsole = false;
    logFileName =
      loggingEnv.logDir + "/spy_relay." + new Date().toISOString() + ".log";
  }

  let logLevel = loggingEnv.logLevel || "info";

  let transport: any;
  if (useConsole) {
    console.log("spy_relay is logging to the console at level [%s]", logLevel);

    transport = new winston.transports.Console({
      level: logLevel,
    });
  } else {
    console.log(
      "spy_relay is logging to [%s] at level [%s]",
      logFileName,
      logLevel
    );

    transport = new winston.transports.File({
      filename: logFileName,
      level: logLevel,
    });
  }

  const logConfiguration: winston.LoggerOptions = {
    // NOTE: do not specify labels in defaultMeta, as it cannot be overridden
    transports: [transport],
    format: winston.format.combine(
      winston.format.splat(),
      winston.format.simple(),
      winston.format.timestamp({
        format: "YYYY-MM-DD HH:mm:ss.SSS",
      }),
      winston.format.errors({ stack: true }),
      winston.format.printf(
        (info: any) =>
          `${[info.timestamp]}|${info.level}|${
            info.labels && info.labels.length > 0
              ? info.labels.join("|")
              : "main"
          }: ${info.message}`
      )
    ),
  };

  return winston.createLogger(logConfiguration);
}
