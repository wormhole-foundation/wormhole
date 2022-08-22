import { beforeAll, test } from "@jest/globals";
import { getLogger, getScopedLogger } from "./logHelper";

// TODO: mock and confirm output

beforeAll(() => {
  require("./loadConfig");
  process.env.LOG_DIR = ".";
});

test("should log default logs", () => {
  const logger = getLogger();
  logger.info("test");
});
test("should use child labels", () => {
  getLogger().child({}).info("test without labels");
  getLogger().child({ labels: [] }).info("test with empty labels");
  getLogger()
    .child({ labels: ["one"] })
    .info("test with one label");
  getLogger()
    .child({ labels: ["one", "two"] })
    .info("test with two labels");
  getLogger()
    .child({ labels: ["one", "two", "three"] })
    .info("test with three labels");
});
test("should allow child label override", () => {
  const root = getLogger();
  const parent = root.child({ labels: ["override-me"] });
  const child = root.child({ labels: ["overridden"] });
  root.info("root log");
  parent.info("parent log");
  child.info("child log");
});
test("scoped logger", () => {
  getScopedLogger([]).info("no labels");
  getScopedLogger(["one"]).info("one label");
});
test("scoped logger inheritance", () => {
  const parent = getScopedLogger(["parent"]);
  const child = getScopedLogger(["child"], parent);
  parent.info("parent log");
  child.info("child log");
});
