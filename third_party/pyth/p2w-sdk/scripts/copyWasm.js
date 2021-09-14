const find = require("find");
const fs = require("fs");

const SOURCE_ROOT = "src";
const TARGET_ROOT = "lib";

find.eachfile(/\.wasm(\..*)?/, SOURCE_ROOT, file => {

    copy = file.replace(SOURCE_ROOT, TARGET_ROOT);

    console.log("copyWasm:", file, "to", copy);

    fs.copyFileSync(file, copy);
});
