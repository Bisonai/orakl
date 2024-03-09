// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console2, console} from "forge-std/Test.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Aggregator} from "../src/Aggregator.sol";

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

    function test_AddAndRemoveOracle() public {
        submissionProxy.addOracle(address(0));
        uint256 oracleLength = submissionProxy.getOracles().length;
        assertEq(oracleLength, 1);

        submissionProxy.removeOracle(address(0));
        oracleLength = submissionProxy.getOracles().length;
        assertEq(oracleLength, 0);
    }

    function test_BatchSubmission() public {
	address offChainSubmissionProxyReporter = address(0);
	address offChainAggregatorReporter = address(1);

        submissionProxy.addOracle(offChainSubmissionProxyReporter);

        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](2);
        uint256 singleSubmitGas;
        uint256 batchSubmitGas;
        address[] memory aggregators = new address[](50);
        int256[] memory submissions = new int256[](50);
        uint256 startGas;

	// multiple single submissions
        for (uint256 i = 0; i < 50; i++) {
            Aggregator aggregator = new Aggregator(timeout, decimals, description);

            oracleAdd[0] = address(submissionProxy);
            oracleAdd[1] = offChainAggregatorReporter;
            aggregator.changeOracles(oracleRemove, oracleAdd, 1, 1, 0);

            aggregators[i] = address(aggregator);
            submissions[i] = 10;

            startGas = gasleft();
            vm.prank(offChainAggregatorReporter);
            aggregator.submit(10);
            singleSubmitGas += estimateGasCost(startGas);
        }

	// single batch submission
        startGas = gasleft();
        vm.prank(offChainSubmissionProxyReporter);
        submissionProxy.submit(aggregators, submissions);
        batchSubmitGas = estimateGasCost(startGas);

        console.log("single submit", singleSubmitGas, "batch submit", batchSubmitGas);
    }
}
