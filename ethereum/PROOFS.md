# Instructions to Reproduce the KEVM Proofs

## Dependencies

First, install [Forge](https://github.com/foundry-rs/foundry/tree/master/forge) and run

```
make dependencies
```

to install the `forge-std` library in the `lib` folder.

Next, [follow the instructions here](https://github.com/runtimeverification/evm-semantics/) to install KEVM (we recommend the fast installation with kup). All proofs were originally run using the KEVM version corresponding to commit [f5c1795aea0c7d6781c94f0e6d4c434ad3ad1982](https://github.com/runtimeverification/evm-semantics/commit/f5c1795aea0c7d6781c94f0e6d4c434ad3ad1982). To update the installation to a specific KEVM version, run

```
kup update kevm --version <branch name or commit hash>
```

## Reproducing the Proofs

Use the following command to run the proofs using KEVM:

```
./run-kevm.sh
```

The script first builds the tests with `forge build`, then kompiles them into a KEVM specification and runs the prover.

The script symbolically executes all tests added to the `tests` variable. By default, it is set to run all of the tests that have been verified using KEVM. To run a single test at a time, comment out the lines with the other tests. Tests can also be run in parallel by changing the value of the `workers` variable.

The script is set to resume proofs from where they left off. This means that if a proof is interrupted halfway, running the script again will continue from that point. Similarly, if a proof has already completed, running the script again will do nothing and just report the result. To instead restart from the beginning, turn on the `reinit` option in the script. Also note that if changes are made to the Solidity code, or if you switch to a different KEVM version, you should turn on the `regen` option the next time you run the script.

**Note:** When building the tests, the `run-kevm.sh` script uses the command `forge build --skip Migrator.sol` to avoid building the `Migrator` contract. The reason is that this contract contains a function named `claim`, which currently causes problems because `claim` is a reserved keyword in KEVM (none of the proofs depend on this contract, but KEVM generates definitions for all contracts built by `forge build`). Until this issue is fixed, if `forge build` has previously been run without the `--skip Migrator.sol` option, it is necessary to delete the `out/Migrator.sol` directory before running `run-kevm.sh`. Otherwise, the script will produce the following error:
```
[Error] Inner Parser: Parse error: unexpected end of file following token '.'.
	Source(/path/to/wormhole/ethereum/out/kompiled/foundry.k)
	Location(7633,23,7633,23)
	7633 |	    rule  ( Migrator . claim ( V0__amount : uint256 ) => #abiCallData (
"claim" , #uint256 ( V0__amount ) , .TypedArgs ) )
	     .	                      ^
[Error] Inner Parser: Parse error: unexpected token 'uint256' following token
':'.
	Source(/path/to/wormhole/ethereum/out/kompiled/foundry.k)
	Location(7633,45,7633,52)
	7633 |	    rule  ( Migrator . claim ( V0__amount : uint256 ) => #abiCallData (
"claim" , #uint256 ( V0__amount ) , .TypedArgs ) )
	     .	                                            ^~~~~~~
[Error] Compiler: Had 2 parsing errors.
```