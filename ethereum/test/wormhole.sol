// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "truffle/Assert.sol";
import "../contracts/Messages.sol";

contract TestMessages is Messages {
  function testQuorum() public { 
    Assert.equal(quorum(0), 1, "it should return quorum");
    Assert.equal(quorum(1), 1, "it should return quorum");
    Assert.equal(quorum(2), 2, "it should return quorum");
    Assert.equal(quorum(3), 3, "it should return quorum");
    Assert.equal(quorum(4), 3, "it should return quorum");
    Assert.equal(quorum(5), 4, "it should return quorum");
    Assert.equal(quorum(6), 5, "it should return quorum");
    Assert.equal(quorum(7), 5, "it should return quorum");
    Assert.equal(quorum(8), 6, "it should return quorum");
    Assert.equal(quorum(9), 7, "it should return quorum");
    Assert.equal(quorum(10), 7, "it should return quorum");
    Assert.equal(quorum(11), 8, "it should return quorum");
    Assert.equal(quorum(12), 9, "it should return quorum");
    Assert.equal(quorum(20), 14, "it should return quorum");
  }  
}