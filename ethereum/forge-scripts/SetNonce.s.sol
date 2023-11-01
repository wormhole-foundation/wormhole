// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {Implementation} from "../contracts/Implementation.sol";
import {Setup} from "../contracts/Setup.sol";
import "forge-std/Script.sol";

contract SetNonce is Script {
    function setNonce(uint64 num) public {
        vm.setNonce(msg.sender, num);
    }

    function incrementNonce(uint64 num) public {
        vm.setNonce(msg.sender, vm.getNonce(msg.sender) + num);
    }
}
