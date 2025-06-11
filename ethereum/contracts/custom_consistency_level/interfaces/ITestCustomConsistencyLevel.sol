// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

interface ITestCustomConsistencyLevel {
    function publishMessage(
        string memory str
    ) external payable returns (uint64 sequence);
}
