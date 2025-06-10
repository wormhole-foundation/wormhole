// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

interface ITestCustomConsistencyLevel {
    function configure() external;

    function publishMessage(
        bytes memory payload
    ) external payable returns (uint64 sequence);
}
