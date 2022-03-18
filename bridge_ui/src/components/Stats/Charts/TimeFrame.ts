import { DurationLike } from "luxon";
import { formatTickDay, formatTickMonth } from "./utils";

export interface TimeFrame {
  interval?: number;
  duration?: DurationLike;
  tickFormatter: (value: any, index: number) => string;
}

export const TIME_FRAMES: { [key: string]: TimeFrame } = {
  "7 days": {
    duration: { days: 7 },
    tickFormatter: formatTickDay,
  },
  "30 days": {
    duration: { days: 30 },
    tickFormatter: formatTickDay,
  },
  "3 months": {
    duration: { months: 3 },
    tickFormatter: formatTickDay,
  },
  "6 months": {
    duration: { months: 6 },
    interval: 30,
    tickFormatter: formatTickMonth,
  },
  "1 year": {
    duration: { years: 1 },
    interval: 30,
    tickFormatter: formatTickMonth,
  },
  "All time": { interval: 30, tickFormatter: formatTickMonth },
};
