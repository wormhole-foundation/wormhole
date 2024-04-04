process.env.CI = true;

const info = console.info;
console.info = function (x) {
  if (x !== "secp256k1 unavailable, reverting to browser version") {
    info(x);
  }
};

const warn = console.warn;
console.warn = function (x) {
  if (
    x !==
    "bigint: Failed to load bindings, pure JS will be used (try npm run rebuild?)"
  ) {
    warn(x);
  }
};
