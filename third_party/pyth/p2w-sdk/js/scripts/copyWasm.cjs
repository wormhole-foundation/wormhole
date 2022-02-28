const find = require("find");
const fs = require("fs");
const path = require("path");

const SOURCE_ROOT = "src";
const TARGET_ROOT = "lib";

find.eachfile(/\.wasm(\..*)?/, SOURCE_ROOT, fname => {

    fname_copy = fname.replace(SOURCE_ROOT, TARGET_ROOT);

    console.log("copyWasm:", fname, "to", fname_copy);

    fs.mkdirSync(path.dirname(fname_copy), {recursive: true});

    fs.copyFileSync(fname, fname_copy);
});
