pragma solidity ^0.8.0;

import "truffle/Assert.sol";
import "../contracts/Messages.sol";

contract TestMessages {
  function testQuorum() public { 
    // An array to hold our test cases
    uint[][] testCases = [[0, 1],[1, 1],[2, 2],[3, 3],[4, 3],[5, 4],[6, 5],[7, 5],[8, 6],[9, 7],[10, 7],[11, 8],[12, 9],[20, 14]];

    // Loop through our testCases array and assert that our expectations are met
    for (uint i=0; i<testCases.length; i++) {
        Assert.equal(Messages.quorum(testCases[i][0]), testCases[i][1], "it should return the right number of signatures for quorum");
    }
  }
}