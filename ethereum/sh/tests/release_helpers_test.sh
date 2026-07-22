#!/bin/bash

set -uo pipefail

script_dir=$(CDPATH= cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
sh_dir=$(CDPATH= cd "$script_dir/.." && pwd -P)
repo_root=$(CDPATH= cd "$sh_dir/../.." && pwd -P)

tests=0
failures=0
tmp_root=$(mktemp -d "${TMPDIR:-/tmp}/wormhole-release-helpers.XXXXXX")
trap 'rm -rf -- "$tmp_root"' EXIT

pass() {
  tests=$((tests + 1))
  printf 'ok %d - %s\n' "$tests" "$1"
}

fail() {
  tests=$((tests + 1))
  failures=$((failures + 1))
  printf 'not ok %d - %s\n' "$tests" "$1" >&2
}

assert_status() {
  local expected="$1" actual="$2" description="$3"
  if [ "$actual" -eq "$expected" ]; then
    pass "$description"
  else
    fail "$description (expected status $expected, got $actual)"
  fi
}

assert_absent() {
  local path="$1" description="$2"
  if [ ! -e "$path" ]; then
    pass "$description"
  else
    fail "$description"
  fi
}

assert_contains() {
  local path="$1" expected="$2" description="$3"
  if grep -Fq -- "$expected" "$path"; then
    pass "$description"
  else
    fail "$description"
  fi
}

assert_not_contains() {
  local path="$1" unexpected="$2" description="$3"
  if grep -Fq -- "$unexpected" "$path"; then
    fail "$description"
  else
    pass "$description"
  fi
}

count_nul_arg() {
  local path="$1" expected="$2" argument count=0
  while IFS= read -r -d '' argument; do
    if [ "$argument" = "$expected" ]; then
      count=$((count + 1))
    fi
  done < "$path"
  printf '%s\n' "$count"
}

assert_nul_args() {
  local path="$1" description="$2" argument index=0
  shift 2
  local actual=()
  while IFS= read -r -d '' argument; do
    actual+=("$argument")
  done < "$path"

  if [ "${#actual[@]}" -ne "$#" ]; then
    fail "$description (expected $# arguments, got ${#actual[@]})"
    return
  fi

  for argument in "$@"; do
    if [ "${actual[$index]}" != "$argument" ]; then
      fail "$description (argument $index differs)"
      return
    fi
    index=$((index + 1))
  done
  pass "$description"
}

tree="$tmp_root/tree"
eth="$tree/ethereum"
bin="$eth/test-bin"
verify_bin="$eth/verify-test-bin"
mkdir -p "$eth/sh" "$eth/env" "$eth/build-forge/Implementation.sol" \
  "$eth/build-forge/BridgeImplementation.sol" "$bin" "$verify_bin" \
  "$tree/deployments/testnet" "$tree/deployments/mainnet"
: > "$eth/build-forge/Implementation.sol/Implementation.json"
: > "$eth/build-forge/BridgeImplementation.sol/BridgeImplementation.json"

scripts=(
  deployCoreBridge.sh
  deployCoreBridgeTron.sh
  deployCoreShutdown.sh
  deployCustomConsistencyLevel.sh
  deployDelegatedManagerSet.sh
  deployDelegatedGuardians.sh
  deployDummyContract.sh
  deployNFTBridge.sh
  deployTokenBridge.sh
  deployTokenBridgeShutdown.sh
  devnetInitialization.sh
  registerAllChainsOnTokenBridge.sh
  registerChainsNFTBridge.sh
  registerChainsTokenBridge.sh
  upgrade.sh
  upgrade_all_testnet.sh
)
for script in "${scripts[@]}"; do
  ln -s "$sh_dir/$script" "$eth/sh/$script"
done

# Stubs record argv and emulate only the external behavior used by the helpers.
cat > "$bin/worm" <<'WORM'
#!/bin/bash
printf 'worm' >> "$TOOL_LOG"
printf ' <%s>' "$@" >> "$TOOL_LOG"
printf '\n' >> "$TOOL_LOG"

if [ "$1" = info ] && [ "$2" = rpc ]; then
  [ "${WORM_RPC_FAIL:-0}" = 1 ] && exit 1
  printf 'https://%s.rpc.invalid\n' "$4"
elif [ "$1" = info ] && [ "$2" = chain-id ]; then
  case "$3" in
    base) printf '30\n' ;;
    demo) printf '999\n' ;;
    blast) printf '36\n' ;;
    ethereum) printf '2\n' ;;
    *) printf '1000\n' ;;
  esac
elif [ "$1" = evm ] && [ "$2" = info ]; then
  [ "${WORM_NO_DEPLOYMENT:-0}" = 1 ] && exit 1
  if [ "${WORM_INVALID_IMPLEMENTATION:-0}" = 1 ]; then
    printf 'not-an-address\n'
    exit 0
  fi
  printf '0x1111111111111111111111111111111111111111\n'
elif [ "$1" = evm ] && [ "$2" = chains ]; then
  [ "${WORM_CHAINS_FAIL:-0}" = 1 ] && exit 1
  [ "${WORM_CHAINS_EMPTY:-0}" = 1 ] && exit 0
  printf 'alpha\nbeta\n'
elif [ "$1" = generate ] && [ "$2" = upgrade ]; then
  [ "${WORM_GENERATE_FAIL:-0}" = 1 ] && exit 1
  [ "${WORM_GENERATE_EMPTY:-0}" = 1 ] && exit 0
  printf 'test-upgrade-vaa\n'
elif [ "$1" = submit ]; then
  [ "${WORM_SUBMIT_FAIL:-0}" = 1 ] && exit 1
  [ -z "${2:-}" ] && exit 1
  exit 0
else
  exit 2
fi
WORM

cat > "$bin/cast" <<'CAST'
#!/bin/bash
printf 'cast' >> "$TOOL_LOG"
printf ' <%s>' "$@" >> "$TOOL_LOG"
printf '\n' >> "$TOOL_LOG"

[ "${CAST_FAIL:-0}" = 1 ] && exit 1
if [ "${CAST_INVALID:-0}" = 1 ]; then
  printf 'not-a-chain-id\n'
  exit 0
fi
rpc_url=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = --rpc-url ]; then
    rpc_url="$2"
    break
  fi
  shift
done
case "$rpc_url" in
  *alpha*) printf '111\n' ;;
  *beta*) printf '222\n' ;;
  *reviewed*) printf '333\n' ;;
  *) printf '444\n' ;;
esac
CAST

cat > "$bin/forge" <<'FORGE'
#!/bin/bash
printf '%s\0' "$@" >> "$FORGE_LOG"
[ "${FORGE_FAIL:-0}" = 1 ] && exit 1

target="$2"
solfile="${target#./forge-scripts/}"
solfile="${solfile%%:*}"
rpc_url=""
arguments=("$@")
index=0
while [ "$index" -lt "${#arguments[@]}" ]; do
  if [ "${arguments[index]}" = --rpc-url ]; then
    index=$((index + 1))
    rpc_url="${arguments[index]}"
    break
  fi
  index=$((index + 1))
done
[ -n "${FORGE_FAIL_RPC_CONTAINS:-}" ] \
  && [[ "$rpc_url" == *"$FORGE_FAIL_RPC_CONTAINS"* ]] \
  && exit 1
case "$rpc_url" in
  *alpha*) chain_id=111 ;;
  *beta*) chain_id=222 ;;
  *reviewed*) chain_id=333 ;;
  *demo*) chain_id=444 ;;
  *) chain_id="${INIT_EVM_CHAIN_ID:-${EVM_CHAIN_ID:-1337}}" ;;
esac
mkdir -p "broadcast/$solfile/$chain_id"
cat > "broadcast/$solfile/$chain_id/run-latest.json" <<'JSON'
{"returns":{"deployedAddress":{"value":"0x2222222222222222222222222222222222222222"},"setupAddress":{"value":"0x2222222222222222222222222222222222222222"},"implAddress":{"value":"0x2222222222222222222222222222222222222222"},"tokenImplementationAddress":{"value":"0x2222222222222222222222222222222222222222"},"bridgeSetupAddress":{"value":"0x2222222222222222222222222222222222222222"},"bridgeImplementationAddress":{"value":"0x2222222222222222222222222222222222222222"},"nftImplementationAddress":{"value":"0x2222222222222222222222222222222222222222"},"implementationAddress":{"value":"0x2222222222222222222222222222222222222222"},"deployedDelegatedGuardians":{"value":"0x2222222222222222222222222222222222222222"}}}
JSON
FORGE

cat > "$bin/jq" <<'JQ'
#!/bin/bash
input_file=""
for argument in "$@"; do
  [ -f "$argument" ] && input_file=$argument
done
if [ -n "$input_file" ]; then
  cat "$input_file" >/dev/null
else
  cat >/dev/null
fi
[ "${JQ_EMPTY:-0}" = 1 ] && exit 0
if [ "${JQ_NULL:-0}" = 1 ]; then
  printf 'null\n'
  exit 0
fi
if [ "${JQ_INVALID:-0}" = 1 ]; then
  printf 'not-an-address\n'
  exit 0
fi
printf '0x2222222222222222222222222222222222222222\n'
JQ

cat > "$bin/npm" <<'NPM'
#!/bin/bash
printf 'npm' >> "$TOOL_LOG"
printf ' <%s>' "$@" >> "$TOOL_LOG"
printf '\n' >> "$TOOL_LOG"
NPM

cat > "$bin/node" <<'NODE'
#!/bin/bash
printf 'node' >> "$TOOL_LOG"
printf ' <%s>' "$@" >> "$TOOL_LOG"
printf '\n' >> "$TOOL_LOG"
NODE

cat > "$verify_bin/curl" <<'CURL'
#!/bin/bash
case "${VERIFY_CURL_MODE:-fail}" in
  match) printf '{"result":"0x00"}\n' ;;
  mismatch) printf '{"result":"0x01"}\n' ;;
  null) printf '{"result":null}\n' ;;
  *) exit 7 ;;
esac
CURL

cat > "$eth/verify" <<'VERIFY'
#!/bin/bash
printf 'verify' >> "$TOOL_LOG"
printf ' <%s>' "$@" >> "$TOOL_LOG"
printf '\n' >> "$TOOL_LOG"

last=""
for argument in "$@"; do
  last="$argument"
done
if [ "${VERIFY_MODE:-}" = skip ]; then
  exit 0
fi
if [ "${VERIFY_MODE:-}" = operational-fail ]; then
  exit 2
fi
if [[ ! "$last" =~ ^0x[0-9A-Fa-f]{40}$ ]]; then
  exit 2
fi
if [ "$last" = 0x2222222222222222222222222222222222222222 ]; then
  [ "${VERIFY_MODE:-}" = post-fail ] && exit 1
  exit 0
fi
exit 1
VERIFY

chmod +x "$bin/worm" "$bin/cast" "$bin/forge" "$bin/jq" "$bin/npm" \
  "$bin/node" "$verify_bin/curl" "$eth/verify"

cat > "$tree/deployments/testnet/tokenBridgeVAAs.csv" <<'CSV'
# chain,vaa
Demo (999) Testnet Token Bridge,0011
Other (1000) Testnet Token Bridge,aabb
CSV
cp "$tree/deployments/testnet/tokenBridgeVAAs.csv" \
  "$tree/deployments/mainnet/tokenBridgeVAAs.csv"

# Any accidental env sourcing creates this harmless marker.
marker="$eth/repository-env-executed"
cat > "$eth/.env" <<ENV
touch "$marker"
MNEMONIC=attacker-controlled
RPC_URL=https://attacker.invalid
REGISTER_OTHER_TOKEN_BRIDGE_VAA=0011
REGISTER_OTHER_NFT_BRIDGE_VAA=0022
ENV
cp "$eth/.env" "$eth/env/.env.demo.testnet"

path="$bin:$PATH"
tool_log="$eth/tool.log"
forge_log="$eth/forge.args"

# Static security and syntax coverage.
if grep -nE '^[[:space:]]*(source|\.)[[:space:]]+.*(\.env|ENV_FILE|env_file)' "$sh_dir"/*.sh >/dev/null; then
  fail 'EVM helpers contain no direct env-file execution'
else
  pass 'EVM helpers contain no direct env-file execution'
fi

syntax_status=0
for script in "${scripts[@]}"; do
  /bin/bash -n "$sh_dir/$script" || syntax_status=1
done
/bin/bash -n "$repo_root/ethereum/verify" || syntax_status=1
assert_status 0 "$syntax_status" 'all covered helpers pass Bash syntax checking'

node_syntax_status=0
node --check "$sh_dir/deployCoreBridgeTron.js" >/dev/null 2>&1 || node_syntax_status=1
assert_status 0 "$node_syntax_status" 'Tron deployment JavaScript passes syntax checking'

# Exercise the real verifier's match, mismatch, and operational-error contract.
verify_artifact="$eth/build-forge/Implementation.sol/verify-test.json"
printf '{"deployedBytecode":{"object":"0x00"}}\n' > "$verify_artifact"
(
  PATH="$verify_bin:$PATH" "$repo_root/ethereum/verify" \
    -r http://local.invalid -c demo \
    "$verify_artifact" 0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 2 "$status" 'verify distinguishes an RPC failure from a bytecode mismatch'

for verify_mode in match mismatch null; do
  (
    PATH="$verify_bin:$PATH" VERIFY_CURL_MODE=$verify_mode \
      "$repo_root/ethereum/verify" \
      -r http://local.invalid -c demo \
      "$verify_artifact" 0x1111111111111111111111111111111111111111
  ) >/dev/null 2>&1
  status=$?
  case "$verify_mode" in
    match) assert_status 0 "$status" 'verify returns success for matching bytecode' ;;
    mismatch) assert_status 1 "$status" 'verify reserves status 1 for bytecode mismatch' ;;
    null) assert_status 2 "$status" 'verify treats a null RPC result as an operational failure' ;;
  esac
done

(
  PATH="$verify_bin:$PATH" VERIFY_CURL_MODE=mismatch \
    "$repo_root/ethereum/verify" \
    -r http://local.invalid -c demo \
    "$eth/build-forge/Implementation.sol/missing.json" \
    0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 2 "$status" 'verify treats a missing build artifact as an operational failure'

# Direct registration keeps the original Forge argv while ignoring .env.
: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret \
    RPC_URL='https://reviewed.invalid/path?a=1&b=2' \
    TOKEN_BRIDGE_ADDRESS=0x1111111111111111111111111111111111111111 \
    TOKEN_BRIDGE_REGISTRATION_VAAS='[0x0011,0xaabb]' \
    ./sh/registerChainsTokenBridge.sh
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'token registration accepts operator environment'
[ "$(count_nul_arg "$forge_log" 'https://reviewed.invalid/path?a=1&b=2')" -eq 1 ] \
  && pass 'registration RPC URL remains one argv element' \
  || fail 'registration RPC URL remains one argv element'
[ "$(count_nul_arg "$forge_log" '[0x0011,0xaabb]')" -eq 1 ] \
  && pass 'registration bytes array remains one argv element' \
  || fail 'registration bytes array remains one argv element'
assert_nul_args "$forge_log" 'direct registration passes the exact Forge argv vector' \
  script \
  ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge \
  --sig 'run(address,bytes[])' \
  0x1111111111111111111111111111111111111111 \
  '[0x0011,0xaabb]' \
  --rpc-url 'https://reviewed.invalid/path?a=1&b=2' \
  --private-key test-only-secret \
  --broadcast
assert_absent "$marker" 'direct registration does not execute local .env'

: > "$forge_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret ./sh/registerChainsTokenBridge.sh
) >/dev/null 2>&1
status=$?
assert_status 1 "$status" 'missing operator configuration fails closed'
if [ ! -s "$forge_log" ]; then
  pass 'missing operator configuration prevents Forge invocation'
else
  fail 'missing operator configuration prevents Forge invocation'
fi

# Register-all preserves CSV aggregation and optional Forge arguments.
: > "$forge_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret RPC_URL=https://reviewed.invalid FORGE_ARGS=--slow \
    ./sh/registerAllChainsOnTokenBridge.sh testnet demo \
      0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'register-all accepts operator environment'
[ "$(count_nul_arg "$forge_log" '[0xaabb]')" -eq 1 ] \
  && pass 'register-all preserves VAA aggregation' \
  || fail 'register-all preserves VAA aggregation'
[ "$(count_nul_arg "$forge_log" '--slow')" -eq 1 ] \
  && pass 'register-all preserves FORGE_ARGS expansion' \
  || fail 'register-all preserves FORGE_ARGS expansion'
assert_nul_args "$forge_log" 'register-all passes the exact Forge argv vector' \
  script \
  ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge \
  --sig 'run(address,bytes[])' \
  0x1111111111111111111111111111111111111111 \
  '[0xaabb]' \
  --rpc-url https://reviewed.invalid \
  --private-key test-only-secret \
  --broadcast --slow
assert_absent "$marker" 'register-all does not execute selected repo env'

# Repository CSV text must remain inside the single ABI argument.
printf 'Demo (999) Testnet Token Bridge,0011\nOther (1000) Testnet Token Bridge,aabb --slow --rpc-url https://evil.invalid\n' \
  > "$tree/deployments/testnet/tokenBridgeVAAs.csv"
: > "$forge_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret RPC_URL=https://reviewed.invalid \
    ./sh/registerAllChainsOnTokenBridge.sh testnet demo \
      0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'CSV option-like text remains data'
[ "$(count_nul_arg "$forge_log" '[0xaabb --slow --rpc-url https://evil.invalid]')" -eq 1 ] \
  && pass 'CSV option-like text remains one Forge argument' \
  || fail 'CSV option-like text remains one Forge argument'
[ "$(count_nul_arg "$forge_log" 'https://evil.invalid')" -eq 0 ] \
  && pass 'CSV text cannot replace the reviewed RPC argument' \
  || fail 'CSV text cannot replace the reviewed RPC argument'

# Numeric IDs distinguish overlapping names such as Base and Base Sepolia.
printf 'Base (30) Testnet Token Bridge,aaaa\nBase Sepolia (10004) Testnet Token Bridge,bbbb\nOther (1000) Testnet Token Bridge,cccc\n' \
  > "$tree/deployments/testnet/tokenBridgeVAAs.csv"
: > "$forge_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret RPC_URL=https://reviewed.invalid \
    ./sh/registerAllChainsOnTokenBridge.sh testnet base \
      0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'register-all matches the exact target chain ID'
[ "$(count_nul_arg "$forge_log" '[0xbbbb,0xcccc]')" -eq 1 ] \
  && pass 'register-all keeps Base Sepolia when targeting Base' \
  || fail 'register-all keeps Base Sepolia when targeting Base'

# Missing or duplicate target IDs must fail before Forge.
target_cases=(missing duplicate)
target_rows=(
  'Other (1000) Testnet Token Bridge,cccc'
  $'Base (30) Testnet Token Bridge,aaaa\nBase duplicate (30) Testnet Token Bridge,dddd'
)
for index in "${!target_cases[@]}"; do
  printf '%s\n' "${target_rows[index]}" > "$tree/deployments/testnet/tokenBridgeVAAs.csv"
  : > "$forge_log"
  (
    cd "$eth" || exit 1
    PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
      MNEMONIC=test-only-secret RPC_URL=https://reviewed.invalid \
      ./sh/registerAllChainsOnTokenBridge.sh testnet base \
        0x1111111111111111111111111111111111111111
  ) >/dev/null 2>&1
  status=$?
  assert_status 1 "$status" \
    "register-all rejects a ${target_cases[index]} target chain ID"
  if [ ! -s "$forge_log" ]; then
    pass "${target_cases[index]} target chain data prevents Forge invocation"
  else
    fail "${target_cases[index]} target chain data prevents Forge invocation"
  fi
done

# Restore the normal fixture and verify explicit RPC configuration.
printf '# chain,vaa\nDemo (999) Testnet Token Bridge,0011\nOther (1000) Testnet Token Bridge,aabb\n' \
  > "$tree/deployments/testnet/tokenBridgeVAAs.csv"
: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret \
    ./sh/registerAllChainsOnTokenBridge.sh testnet demo \
      0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 1 "$status" 'register-all requires an explicit RPC URL'
if [ ! -s "$forge_log" ]; then
  pass 'missing register-all RPC prevents Forge invocation'
else
  fail 'missing register-all RPC prevents Forge invocation'
fi

# Verify compatibility with the repository's real CSV files.
cp "$repo_root/deployments/testnet/tokenBridgeVAAs.csv" \
  "$tree/deployments/testnet/tokenBridgeVAAs.csv"
: > "$forge_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret RPC_URL=https://reviewed.invalid \
    ./sh/registerAllChainsOnTokenBridge.sh testnet blast \
      0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'register-all accepts the repository testnet CSV format'

cp "$repo_root/deployments/mainnet/tokenBridgeVAAs.csv" \
  "$tree/deployments/mainnet/tokenBridgeVAAs.csv"
: > "$forge_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-only-secret RPC_URL=https://reviewed.invalid \
    ./sh/registerAllChainsOnTokenBridge.sh mainnet ethereum \
      0x1111111111111111111111111111111111111111
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'register-all accepts the repository mainnet CSV format'

# Standalone deployment helpers continue to consume exported values.
common_deploy_env=(
  PATH="$path"
  TOOL_LOG="$tool_log"
  FORGE_LOG="$forge_log"
  INIT_EVM_CHAIN_ID=1337
  MNEMONIC=test-only-secret
  RPC_URL=https://reviewed.invalid
  FORGE_ARGS=
)
for helper in deployCoreShutdown.sh deployTokenBridgeShutdown.sh; do
  (
    cd "$eth" || exit 1
    env "${common_deploy_env[@]}" "./sh/$helper"
  ) >/dev/null 2>&1
  status=$?
  assert_status 0 "$status" "$helper accepts operator environment"
done

(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    TRON_PRIVATE_KEY=test-only-secret TRON_FULL_HOST=https://reviewed.invalid \
    INIT_SIGNERS='[0x1111111111111111111111111111111111111111]' \
    INIT_CHAIN_ID=2 INIT_GOV_CHAIN_ID=1 INIT_GOV_CONTRACT=0x00 \
    INIT_EVM_CHAIN_ID=1337 ./sh/deployCoreBridgeTron.sh
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'Tron deployment accepts operator environment'
assert_contains "$tool_log" 'node <sh/deployCoreBridgeTron.js>' \
  'Tron deployment reaches its unchanged Node command'

# The complete devnet workflow reads registration VAAs as data but never runs
# the shell command placed in .env.
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    DEV=True CHAIN_ID=2 EVM_CHAIN_ID=1337 FORGE_ARGS= \
    INIT_SIGNERS='[0x1111111111111111111111111111111111111111]' \
    INIT_CHAIN_ID=2 INIT_GOV_CHAIN_ID=1 INIT_GOV_CONTRACT=0x00 \
    INIT_EVM_CHAIN_ID=1337 BRIDGE_INIT_CHAIN_ID=2 \
    BRIDGE_INIT_GOV_CHAIN_ID=1 BRIDGE_INIT_GOV_CONTRACT=0x00 \
    BRIDGE_INIT_WETH=0x1111111111111111111111111111111111111111 \
    BRIDGE_INIT_FINALITY=1 ./sh/devnetInitialization.sh
) >/dev/null 2>&1
status=$?
assert_status 0 "$status" 'devnet workflow remains functional with operator environment'
assert_absent "$marker" 'devnet treats .env as registration data, not shell code'

# Upgrade: testnet deploy/verify/submit, mainnet output, skip, bulk, and failure.
: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-signing-key GUARDIAN_MNEMONIC=test-guardian-key \
    FORGE_ARGS='--slow --gas-estimate-multiplier 130' \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-testnet.out" 2>&1
status=$?
assert_status 0 "$status" 'testnet Core upgrade completes'
assert_contains "$tool_log" 'worm <submit> <test-upgrade-vaa> <-n> <testnet>' \
  'testnet upgrade submits generated governance VAA'
[ "$(count_nul_arg "$forge_log" '--gas-estimate-multiplier')" -eq 1 ] \
  && pass 'upgrade preserves operator FORGE_ARGS' \
  || fail 'upgrade preserves operator FORGE_ARGS'
assert_nul_args "$forge_log" 'upgrade passes the exact Forge argv vector' \
  script \
  ./forge-scripts/DeployCoreImplementationOnly.s.sol:DeployCoreImplementationOnly \
  --rpc-url https://demo.rpc.invalid \
  --private-key test-signing-key \
  --broadcast --slow --gas-estimate-multiplier 130
assert_absent "$marker" 'upgrade does not execute selected repo env'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-mainnet-key RPC_URL=https://reviewed.rpc.invalid \
    ./sh/upgrade.sh mainnet TokenBridge demo
) > "$eth/upgrade-mainnet.out" 2>&1
status=$?
assert_status 0 "$status" 'mainnet TokenBridge upgrade completes without guardian key'
assert_contains "$eth/upgrade-mainnet.out" \
  '../scripts/contract-upgrade-governance.sh -c demo -m token_bridge -a 0x2222222222222222222222222222222222222222' \
  'mainnet upgrade prints the governance command'
assert_not_contains "$tool_log" 'worm <submit>' \
  'mainnet upgrade does not submit testnet governance'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian VERIFY_MODE=skip \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-skip.out" 2>&1
status=$?
assert_status 0 "$status" 'matching implementation skips upgrade successfully'
if [ ! -s "$forge_log" ]; then
  pass 'matching implementation does not invoke Forge'
else
  fail 'matching implementation does not invoke Forge'
fi

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian VERIFY_MODE=operational-fail \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-verification-error.out" 2>&1
status=$?
assert_status 2 "$status" 'operational verification failure stops the upgrade'
if [ ! -s "$forge_log" ]; then
  pass 'operational verification failure prevents deployment'
else
  fail 'operational verification failure prevents deployment'
fi

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-bulk-key GUARDIAN_MNEMONIC=test-bulk-guardian \
    CHAINS='alpha beta' MODULES=Core ./sh/upgrade_all_testnet.sh
) > "$eth/upgrade-bulk.out" 2>&1
status=$?
assert_status 0 "$status" 'bulk upgrade handles two chains'
assert_contains "$tool_log" 'cast <chain-id> <--rpc-url> <https://alpha.rpc.invalid>' \
  'bulk upgrade derives the alpha chain ID'
assert_contains "$tool_log" 'cast <chain-id> <--rpc-url> <https://beta.rpc.invalid>' \
  'bulk upgrade derives the beta chain ID'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian \
    RPC_URL=https://reviewed.rpc.invalid CHAINS='alpha beta' MODULES=Core \
    ./sh/upgrade_all_testnet.sh
) > "$eth/upgrade-multi-chain-rpc.out" 2>&1
status=$?
assert_status 0 "$status" 'bulk upgrade ignores an inherited RPC override'
assert_not_contains "$tool_log" 'cast <chain-id> <--rpc-url> <https://reviewed.rpc.invalid>' \
  'bulk upgrade does not reuse one inherited RPC across chains'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-bulk-key GUARDIAN_MNEMONIC=test-bulk-guardian \
    CHAINS='alpha beta' MODULES=Core FORGE_FAIL_RPC_CONTAINS=alpha \
    ./sh/upgrade_all_testnet.sh
) > "$eth/upgrade-bulk-partial-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'bulk upgrade reports a partial failure'
assert_not_contains "$tool_log" 'cast <chain-id> <--rpc-url> <https://beta.rpc.invalid>' \
  'bulk upgrade stops after the first failed child'

: > "$tool_log"
(
  cd "$eth" || exit 1
  unset CHAINS MODULES
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_CHAINS_FAIL=1 \
    ./sh/upgrade_all_testnet.sh
) > "$eth/upgrade-chain-enumeration-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'bulk upgrade rejects failed chain enumeration'
assert_not_contains "$tool_log" 'worm <evm> <info>' \
  'failed chain enumeration prevents upgrade attempts'

: > "$tool_log"
(
  cd "$eth" || exit 1
  unset CHAINS MODULES
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_CHAINS_EMPTY=1 \
    ./sh/upgrade_all_testnet.sh
) > "$eth/upgrade-empty-chain-list.out" 2>&1
status=$?
assert_status 1 "$status" 'bulk upgrade rejects an empty chain list'
assert_not_contains "$tool_log" 'worm <evm> <info>' \
  'empty chain enumeration prevents upgrade attempts'

: > "$tool_log"
(
  cd "$eth" || exit 1
  unset CHAINS MODULES
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian VERIFY_MODE=skip \
    ./sh/upgrade_all_testnet.sh
) > "$eth/upgrade-default-modules.out" 2>&1
status=$?
assert_status 0 "$status" 'bulk upgrade default modules are supported'
assert_not_contains "$tool_log" '<NFTBridge>' \
  'bulk upgrade defaults exclude unsupported NFTBridge'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key RPC_URL=https://reviewed.rpc.invalid CAST_INVALID=1 \
    ./sh/upgrade.sh mainnet Core demo
) > "$eth/upgrade-invalid-chain.out" 2>&1
status=$?
assert_status 1 "$status" 'invalid RPC chain ID fails closed'
if [ ! -s "$forge_log" ]; then
  pass 'invalid RPC chain ID prevents Forge invocation'
else
  fail 'invalid RPC chain ID prevents Forge invocation'
fi

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key ./sh/upgrade.sh typo Core demo
) > "$eth/upgrade-invalid-network.out" 2>&1
status=$?
assert_status 1 "$status" 'invalid network fails closed'
if [ ! -s "$tool_log" ] && [ ! -s "$forge_log" ]; then
  pass 'invalid network fails before external commands'
else
  fail 'invalid network fails before external commands'
fi

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_RPC_FAIL=1 \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-rpc-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'RPC lookup failure fails closed'
assert_not_contains "$tool_log" 'cast <chain-id>' \
  'RPC lookup failure prevents chain-ID lookup'
if [ ! -s "$forge_log" ]; then
  pass 'RPC lookup failure prevents Forge invocation'
else
  fail 'RPC lookup failure prevents Forge invocation'
fi

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian FORGE_FAIL=1 \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-forge-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'Forge failure stops the upgrade'
assert_not_contains "$tool_log" 'worm <submit>' \
  'Forge failure prevents governance submission'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian VERIFY_MODE=post-fail \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-verification-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'post-deployment verification failure stops the upgrade'
assert_not_contains "$tool_log" 'worm <submit>' \
  'post-deployment verification failure prevents governance submission'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_GENERATE_FAIL=1 \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-generation-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'governance generation failure stops the upgrade'
assert_not_contains "$tool_log" 'worm <submit>' \
  'governance generation failure prevents submission'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_GENERATE_EMPTY=1 \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-empty-governance.out" 2>&1
status=$?
assert_status 1 "$status" 'empty generated governance VAA stops the upgrade'
assert_contains "$tool_log" 'worm <submit> <> <-n> <testnet>' \
  'empty generated governance VAA is rejected by submission'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_SUBMIT_FAIL=1 \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-submission-failure.out" 2>&1
status=$?
assert_status 1 "$status" 'governance submission failure is reported'

: > "$forge_log"
: > "$tool_log"
(
  cd "$eth" || exit 1
  PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
    MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian WORM_INVALID_IMPLEMENTATION=1 \
    ./sh/upgrade.sh testnet Core demo
) > "$eth/upgrade-invalid-current-address.out" 2>&1
status=$?
assert_status 2 "$status" 'malformed current implementation address fails closed'
if [ ! -s "$forge_log" ]; then
  pass 'malformed current implementation prevents Forge invocation'
else
  fail 'malformed current implementation prevents Forge invocation'
fi

for jq_mode in JQ_EMPTY JQ_NULL JQ_INVALID; do
  : > "$forge_log"
  : > "$tool_log"
  (
    cd "$eth" || exit 1
    export "$jq_mode=1"
    PATH="$path" TOOL_LOG="$tool_log" FORGE_LOG="$forge_log" \
      MNEMONIC=test-key GUARDIAN_MNEMONIC=test-guardian \
      ./sh/upgrade.sh testnet Core demo
  ) > "$eth/upgrade-${jq_mode}.out" 2>&1
  status=$?
  assert_status 1 "$status" "$jq_mode deployment address fails closed"
  assert_not_contains "$tool_log" 'worm <submit>' \
    "$jq_mode deployment address prevents governance submission"
done

printf '1..%d\n' "$tests"
if [ "$failures" -ne 0 ]; then
  printf '%d test(s) failed\n' "$failures" >&2
  exit 1
fi
printf '%d tests passed\n' "$tests"
