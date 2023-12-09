import hashlib
from os import system, environ
from typing import TypedDict

from pyteal import (
    compileTeal,
    Mode,
    Expr,
    OptimizeOptions,
)

class AssemblyResult(TypedDict):
    # Bytecode of the TEAL program
    result: bytes
    # TEAL -> bytecode map
    symbol_map: str
    # SHA512_256 hash of the program
    hash: str

def fullyCompileContract(genTeal, contract: Expr, name, devmode) -> AssemblyResult:
    if genTeal:
        if devmode:
            teal = compileTeal(contract, mode=Mode.Application, version=6, assembleConstants=True)
        else:
            teal = compileTeal(contract, mode=Mode.Application, version=6, assembleConstants=True, optimize=OptimizeOptions(scratch_slots=True))

        with open(name, "w") as f:
            print("Writing " + name)
            f.write(teal)
    else:
        with open(name, "r") as f:
            print("Reading " + name)
            teal = f.read()

    goalBin = environ.get("ALGORAND_GOAL_BIN", "goal")
    status = system(f"{goalBin} clerk compile --map --outfile '{name + '.bin'}' '{name}' ")
    if status != 0:
        raise Exception("Failed to compile")

    with open(name + ".bin", "rb") as contractBin:
        with open(name + ".hash", "w") as fout:
            binary = contractBin.read()
            hash = hashProgram(binary)
            fout.write(hash)

    with open(name + '.bin.map', "r") as mapFile:
        symbol_map = mapFile.read()

    return { "result": binary, "hash": hash, "symbol_map": symbol_map }

def hashProgram(program: bytes) -> str:
    checksum = hashlib.new("sha512_256")
    checksum.update(b"Program")
    checksum.update(program)
    return checksum.hexdigest()
