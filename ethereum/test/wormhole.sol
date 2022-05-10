pragma solidity ^0.8.0;

import "truffle/Assert.sol";
import "../contracts/Messages.sol";

contract TestMessages {
  function testQuorum() { 
    // An array to hold our test cases
    uint[2][] testCases;

    // Define all the testCases and add them to array
    testCases.push([0, 1]);
    testCases.push([1, 1]);
    testCases.push([2, 2]);
    testCases.push([3, 3]);
    testCases.push([4, 3]);
    testCases.push([5, 4]);
    testCases.push([6, 5]);
    testCases.push([7, 5]);
    testCases.push([8, 6]);
    testCases.push([9, 7]);
    testCases.push([10, 7]);
    testCases.push([11, 8]);
    testCases.push([12, 9]);
    testCases.push([20, 14]);
    testCases.push([25, 17]);

    // Loop through our testCases array and assert that our expectations are met
    for (uint i=0; i<testCases.length; i++) {
        Assert.equal(quorum(testCases[i][0]), testCases[i][1], "it should return the right number of signatures for quorum");
    }
  }
}