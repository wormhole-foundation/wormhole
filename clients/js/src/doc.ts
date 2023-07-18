import * as fs from "fs";

import yargs from "yargs";
// Side effects are here to trigger before the afflicted libraries' on-import warnings can be emitted.
// It is also imported so that it can side-effect without being tree-shaken.
import "./side-effects";
// https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
import * as aptos from "./cmds/aptos";
import * as editVaa from "./cmds/editVaa";
import * as evm from "./cmds/evm";
import * as generate from "./cmds/generate";
import * as info from "./cmds/info";
import * as near from "./cmds/near";
import * as parse from "./cmds/parse";
import * as recover from "./cmds/recover";
import * as submit from "./cmds/submit";
import * as sui from "./cmds/sui";
import * as verifyVaa from "./cmds/verifyVaa";
import * as status from "./cmds/status";


const MD_TAG = "<!--CLI_USAGE-->";

async function getHelpText(cmd: any): Promise<string> {
  // Note that `yargs` is called as a function to produce a fresh copy.
  // Otherwise the imported module is effectively a singleton where state from 
  // other commands is accumulated from repeat calls.
  return await cmd.builder(yargs()).scriptName(`worm ${cmd.command}`).getHelp();
}

(async function () {
  const cmds = [
    aptos,
    editVaa,
    evm,
    generate,
    info,
    near,
    parse,
    recover,
    submit,
    sui,
    verifyVaa,
    status
  ];

  const helpOutputs: Buffer[] = [];
  for (const cmd of cmds) {
    const helpText = await getHelpText(cmd);

    helpOutputs.push(Buffer.from(`
<details>
<summary> ${cmd.command} </summary>

\`\`\`sh
${helpText}
\`\`\`
</details>
`))
  }



  const f = fs.readFileSync("README.md");
  const startIdx = f.indexOf(MD_TAG, 0);
  const stopIdx = f.indexOf(MD_TAG, startIdx + 1);

  const head = f.subarray(0, startIdx + MD_TAG.length);
  const tail = f.subarray(stopIdx, f.length);

  const content = Buffer.concat([head, ...helpOutputs, tail])

  fs.writeFileSync("README.md", content.toString())
})();
