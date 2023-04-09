import * as winston from "winston";

export const rootLogger = winston.createLogger({
  transports: [
    new winston.transports.Console({
      level: "info",
    }),
  ],
  format: winston.format.combine(
    winston.format.colorize(),
    winston.format.splat(),
    winston.format.simple(),
    winston.format.timestamp({
      format: "YYYY-MM-DD HH:mm:ss.SSS",
    }),
    winston.format.errors({ stack: true })
  ),
});
