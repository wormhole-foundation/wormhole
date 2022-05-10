pragma solidity ^0.8.0;

import "truffle/Assert.sol";
import "../contracts/Messages.sol";

contract TestMessages {
  function testQuorum() { 
    // An array to store our test cases
    type[] testCases;

    // A struct to define our test cases
    struct TestCase {
        uint numGuardians;
        uint numRequiredForQuorum;
    }

    // Define all the testCases and add them to array
    testCases.push(TestCase(0, 1));
    testCases.push(TestCase(1, 1));
    testCases.push(TestCase(2, 2));
    testCases.push(TestCase(3, 3));
    testCases.push(TestCase(4, 3));
    testCases.push(TestCase(5, 4));
    testCases.push(TestCase(6, 5));
    testCases.push(TestCase(7, 5));
    testCases.push(TestCase(8, 6));
    testCases.push(TestCase(9, 7));
    testCases.push(TestCase(10, 7));
    testCases.push(TestCase(11, 8));
    testCases.push(TestCase(12, 9));
    testCases.push(TestCase(20, 14));
    testCases.push(TestCase(25, 17));
    testCases.push(TestCase(100, 76));

    // Loop through our testCases array and assert that our expectations are met
    uint testCaseLength = testCases.length;
    for (uint i=0; i<testCaseLength; i++) {
        testCase = testCases[i]
        Assert.equal(quorum(testCase.numGuardians), testCase.numRequiredForQuorum, "it should return the right quorum expectation");
    }
  }
}