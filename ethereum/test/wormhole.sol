pragma solidity ^0.8.0;

import "truffle/Assert.sol";
import "../contracts/Messages.sol";

contract TestMessages {
  function testQuorum() { 
    // An array to store our test cases
    type[] testCases;

    // A struct to define our test cases
    struct TestCase {
        Signature[] signatures;
        GuardianSet guardianSet;
        bool result;
    }

    // All our test cases to the array
    // TODO: figure out how to build up these test cases in Solidity
    testCases.push(TestCase(signatures1, guardianSet, true));
    testCases.push(TestCase(signatures2, guardianSet, false));

    // Loop through our test cases and make sure they meet our expectations
    uint testCaseLength = testCases.length;
    for (uint i=0; i<testCaseLength; i++) {
        testCase = testCases[i]
        Assert.equal(quorum(testCase.signatures, testCase.guardianSet), testCase.result, "We should return the right quorum expectation");
    }
  }
}