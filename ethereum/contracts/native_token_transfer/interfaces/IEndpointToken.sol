// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

interface IEndpointToken {
    function mint(address account, uint256 amount) external;

    function burnFrom(address account, uint256 amount) external;
}
