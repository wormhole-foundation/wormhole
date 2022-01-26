import os
import socketserver
import subprocess
import sys

# Settings specific to local devnet Pyth instance
PYTH = os.environ.get("PYTH", "./pyth")
PYTH_KEY_STORE = os.environ.get("PYTH_KEY_STORE", "/home/pyth/.pythd")
PYTH_PROGRAM_KEYPAIR = os.environ.get(
    "PYTH_PROGRAM_KEYPAIR", f"{PYTH_KEY_STORE}/publish_key_pair.json"
)
PYTH_PROGRAM_SO_PATH = os.environ.get("PYTH_PROGRAM_SO", "../target/oracle.so")
PYTH_PUBLISHER_KEYPAIR = os.environ.get(
    "PYTH_PUBLISHER_KEYPAIR", f"{PYTH_KEY_STORE}/publish_key_pair.json"
)
PYTH_PUBLISHER_INTERVAL = float(os.environ.get("PYTH_PUBLISHER_INTERVAL", "5"))

# 0 setting disables airdropping
SOL_AIRDROP_AMT = int(os.environ.get("SOL_AIRDROP_AMT", 0))

# SOL RPC settings
SOL_RPC_HOST = os.environ.get("SOL_RPC_HOST", "solana-devnet")
SOL_RPC_PORT = int(os.environ.get("SOL_RPC_PORT", 8899))
SOL_RPC_URL = os.environ.get(
    "SOL_RPC_URL", "http://{0}:{1}".format(SOL_RPC_HOST, SOL_RPC_PORT)
)

# A TCP port we open when a service is ready
READINESS_PORT = int(os.environ.get("READINESS_PORT", "2000"))


def run_or_die(args, die=True, **kwargs):
    """
    Opinionated subprocess.run() call with fancy logging
    """
    args_readable = " ".join(args)
    print(f"CMD RUN\t{args_readable}", file=sys.stderr)
    sys.stderr.flush()
    ret = subprocess.run(args, text=True, **kwargs)

    if ret.returncode != 0:
        print(f"CMD FAIL {ret.returncode}\t{args_readable}", file=sys.stderr)

        out = ret.stdout if ret.stdout is not None else "<not captured>"
        err = ret.stderr if ret.stderr is not None else "<not captured>"

        print(f"CMD STDOUT\n{out}", file=sys.stderr)
        print(f"CMD STDERR\n{err}", file=sys.stderr)

        if die:
            sys.exit(ret.returncode)
        else:
            print(f'{"CMD DIE FALSE"}', file=sys.stderr)

    else:
        print(f"CMD OK\t{args_readable}", file=sys.stderr)
    sys.stderr.flush()
    return ret


def pyth_run_or_die(subcommand, args=[], debug=False, confirm=True, **kwargs):
    """
    Pyth boilerplate in front of run_or_die
    """
    return run_or_die(
        [PYTH, subcommand] + args + (["-d"] if debug else [])
        # Note: not all pyth subcommands accept -n
        + ([] if confirm else ["-n"])
        + ["-k", PYTH_KEY_STORE]
        + ["-r", SOL_RPC_HOST]
        + ["-c", "finalized"],
        **kwargs,
    )


def sol_run_or_die(subcommand, args=[], **kwargs):
    """
    Solana boilerplate in front of run_or_die
    """
    return run_or_die(["solana", subcommand] + args + ["--url", SOL_RPC_URL], **kwargs)


class ReadinessTCPHandler(socketserver.StreamRequestHandler):
    def handle(self):
        """TCP black hole"""
        self.rfile.read(64)


def readiness():
    """
    Accept connections from readiness probe
    """
    with socketserver.TCPServer(
        ("0.0.0.0", READINESS_PORT), ReadinessTCPHandler
    ) as srv:
        srv.serve_forever()
