import * as fs from "fs/promises";

async function main() {
  const fnames = process.argv.slice(2);

  await Promise.all(fnames.map(async (fname) => {
    console.log(`Erasing types from ${fname}...`);
    const iface = await fs.readFile(fname);
    const erased = eraseTypes(iface.toString());
    await fs.writeFile(fname.replace("Typed.sol", ".sol"), erased);
  }))
  console.log("Done.")
}

function eraseTypes(file: string) {
  const typeMap: Record<string, string> = {
    "Wei ": "uint256 ",
    LocalNative: "uint256",
    TargetNative: "uint256",
    "Gas ": "uint256 ",
    GasPrice: "uint256",
    WeiPrice: "uint256",
    Dollar: "uint256",
    '\nimport "./TypedUnits.sol";\n': "", // delete this import
    "^0.8.19;": "^0.8.0;",
  };

  const escapeRegExp = (str: string) =>
    str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const regex = new RegExp(
    Object.keys(typeMap)
      .map(escapeRegExp)
      .join("|"),
    "g"
  );

  const replacedText = file.replace(regex, (match) => typeMap[match]);

  return replacedText;
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
