import { getCommonEnvironment } from "../configureEnv";

//Be careful not to access this before having called init logger, or it will be undefined
let logger: any;

export function getLogger() {
  if (logger) {
    return logger;
  } else {
    logger = initLogger();
    return logger;
  }
}

function initLogger() {
  const winston = require("winston");
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

  const logConfiguration = {
    transports: [transport],
    format: winston.format.combine(
      winston.format.splat(),
      winston.format.simple(),
      winston.format.timestamp({
        format: "YYYY-MM-DD HH:mm:ss.SSS",
      }),
      winston.format.printf(
        (info: any) => `${[info.timestamp]}|${info.level}|${info.message}`
      )
    ),
  };

  return winston.createLogger(logConfiguration);
}
