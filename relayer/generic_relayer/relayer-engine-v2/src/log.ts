import * as winston from "winston";

export function rootLogger(level = "debug", format: "text" | "json" = "json") {
  return winston.createLogger({
    transports: [
      new winston.transports.Console({
        level,
      }),
    ],
    format: format === "text" ? textFormat : jsonFormat,
  });
}

const textFormat = winston.format.combine(
  winston.format.colorize(),
  winston.format.splat(),
  winston.format.simple(),
  winston.format.timestamp({
    format: "YYYY-MM-DD HH:mm:ss.SSS",
  }),
  winston.format.errors({ stack: true })
);

const jsonFormat = winston.format.combine(
  winston.format.timestamp(),
  winston.format.json(),
  winston.format.errors({ stack: true })
);

type ExecutionContext = {};
