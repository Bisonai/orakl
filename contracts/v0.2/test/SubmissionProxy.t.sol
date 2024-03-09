// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console2, console} from "forge-std/Test.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Aggregator} from "../src/Aggregator.sol";

// TODO test submit after oracle expires
contract SubmissionProxyTest is Test {
    SubmissionProxy submissionProxy;
    uint32 timeout = 10;
    address validator = address(0);
    uint8 decimals = 18;
    string description = "Test Aggregator";
    uint256 timestamp = 1706170779;

    function estimateGasCost(uint256 startGas) internal view returns (uint256) {
        return startGas - gasleft();
    }

    function setUp() public {
        vm.warp(timestamp);
        submissionProxy = new SubmissionProxy();
    }

    // Cannot remove non-existing oracle
    function testFail_RemoveOracle() public {
	submissionProxy.removeOracle(address(0));
    }

    // Add and remove oracle
    function test_AddAndRemoveOracle() public {
        submissionProxy.addOracle(address(0));
        uint256 numOracles = submissionProxy.getOracles().length;
        assertEq(numOracles, 1);

        submissionProxy.removeOracle(address(0));
        numOracles = submissionProxy.getOracles().length;
        assertEq(numOracles, 0);
    }

    function test_BatchSubmission() public {
	uint256 numOracles = 50;
	address offChainSubmissionProxyReporter = address(0);
	address offChainAggregatorReporter = address(1);

        submissionProxy.addOracle(offChainSubmissionProxyReporter);

        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](2);
        uint256 singleSubmissionGas;
        uint256 batchSubmissionGas;
        address[] memory aggregators = new address[](numOracles);
        int256[] memory submissions = new int256[](numOracles);
        uint256 startGas;

	// multiple single submissions
        for (uint256 i = 0; i < numOracles; i++) {
            Aggregator aggregator = new Aggregator(timeout, decimals, description);

            oracleAdd[0] = address(submissionProxy);
            oracleAdd[1] = offChainAggregatorReporter;
            aggregator.changeOracles(oracleRemove, oracleAdd, 1, 1, 0);

            aggregators[i] = address(aggregator);
            submissions[i] = 10;

            vm.prank(offChainAggregatorReporter);
	    aggregator.submit(10); // storage warmup

	    vm.prank(offChainAggregatorReporter);
            startGas = gasleft();
            aggregator.submit(11);
            singleSubmissionGas += estimateGasCost(startGas);
        }

	// single batch submission
	vm.prank(offChainSubmissionProxyReporter);
	submissionProxy.submit(aggregators, submissions); // storage warmup

        vm.prank(offChainSubmissionProxyReporter);
        startGas = gasleft();
        submissionProxy.submit(aggregators, submissions);
        batchSubmissionGas = estimateGasCost(startGas);

        console.log("single submit", singleSubmissionGas, "batch submit", batchSubmissionGas);
	if (singleSubmissionGas > batchSubmissionGas) {
	    console.log("save", singleSubmissionGas - batchSubmissionGas);
	} else {
	    console.log("waste", batchSubmissionGas - singleSubmissionGas);
	}
    }
}
