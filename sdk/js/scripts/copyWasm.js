const fs = require("fs");
fs.copyFileSync(
  "src/solana/core/bridge_bg.wasm",
  "lib/solana/core/bridge_bg.wasm"
);
fs.copyFileSync(
  "src/solana/core/bridge_bg.wasm.d.ts",
  "lib/solana/core/bridge_bg.wasm.d.ts"
);
fs.copyFileSync(
  "src/solana/token/token_bridge_bg.wasm",
  "lib/solana/token/token_bridge_bg.wasm"
);
fs.copyFileSync(
  "src/solana/token/token_bridge_bg.wasm.d.ts",
  "lib/solana/token/token_bridge_bg.wasm.d.ts"
);
