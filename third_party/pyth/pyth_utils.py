import os
import sys
import subprocess

PYTH=os.environ.get("PYTH", "./pyth")
PYTH_KEY_STORE = os.environ.get("PYTH_KEY_STORE", "/home/pyth/.pythd")
PYTH_PROGRAM_KEYPAIR = f"{PYTH_KEY_STORE}/program_key_pair.json"
PYTH_PROGRAM_SO_PATH=os.environ.get("PYTH_PROGRAM_SO", "../target/oracle.so")
PYTH_PUBLISHER_KEYPAIR = f"{PYTH_KEY_STORE}/publish_key_pair.json"
PYTH_PUBLISHER_INTERVAL = float(os.environ.get("PYTH_PUBLISHER_INTERVAL", "5"))

SOL_AIRDROP_AMT = 100
SOL_RPC_HOST = "solana-devnet"
SOL_RPC_PORT = 8899
SOL_RPC_URL = f"http://{SOL_RPC_HOST}:{str(SOL_RPC_PORT)}"

READINESS_PORT=os.environ.get("READINESS_PORT", "2000")

# pretend we're set -e
def run_or_die(args, die=True, **kwargs):
    args_readable = ' '.join(args)
    print(f"CMD RUN\t{args_readable}", file=sys.stderr)
    sys.stderr.flush()
    ret = subprocess.run(args, text=True, **kwargs)

    if ret.returncode is not 0:
        print(f"CMD FAIL {ret.returncode}\t{args_readable}", file=sys.stderr)

        out = ret.stdout if ret.stdout is not None else "<not captured>"
        err = ret.stderr if ret.stderr is not None else "<not captured>"

        print(f"CMD STDOUT\n{out}", file=sys.stderr)
        print(f"CMD STDERR\n{err}", file=sys.stderr)

        if die:
            sys.exit(ret.returncode)
        else:
            print(f"CMD DIE FALSE", file=sys.stderr)

    else:
        print(f"CMD OK\t{args_readable}", file=sys.stderr)
    sys.stderr.flush()
    return ret

# Pyth boilerplate in front of run_or_die
def pyth_run_or_die(subcommand, args=[], debug=False, confirm=True, **kwargs):
    return run_or_die([PYTH, subcommand]
                      + args
                      + (["-d"] if debug else [])
                      + ([] if confirm else ["-n"]) # Note: not all pyth subcommands accept -n
                      + ["-k", PYTH_KEY_STORE]
                      + ["-r", SOL_RPC_HOST]
                      + ["-c", "finalized"], **kwargs)

# Solana boilerplate in front of run_or_die
def sol_run_or_die(subcommand, args=[], **kwargs):
    return run_or_die(["solana", subcommand]
                         + args
                         + ["--url", SOL_RPC_URL], **kwargs)
