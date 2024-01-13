My proposal is to handle error exceptions in a better way and change the semantics a bit:

Example, here is the require in `Governance.sol`:

```
    function submitContractUpgrade(bytes memory _vm) public {
        require(!isFork(), "invalid fork");
    // ... rest of the function
    }
```

This is how I suggest to change it, with if - revert:

```
    if (!isFork()) revert InvalidFork();
```

For that we would need to create a `GovernanceErrors` contract, like [this](https://github.com/luislucena16/wormhole/blob/feature/handle-error-messages/ethereum/contracts/GovernanceErrors.sol)

Note: this is a proposal to be approved or not, that is why I did not make the full modification in the contract `Governance`.