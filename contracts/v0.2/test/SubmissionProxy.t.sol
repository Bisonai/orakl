// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2, console} from "forge-std/Test.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Aggregator} from "../src/Aggregator.sol";

contract SubmissionProxyTest is Test {
    SubmissionProxy batchSubmission;
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
        batchSubmission = new SubmissionProxy();
    }

    function test_AddAndRemoveOracle() public {
        batchSubmission.addOracle(address(0));
        uint256 oracleLength = batchSubmission.getOracles().length;
        assertEq(oracleLength, 1);

        batchSubmission.removeOracle(address(0));
        oracleLength = batchSubmission.getOracles().length;
        assertEq(oracleLength, 0);
    }

    function test_BatchSubmission() public {
        batchSubmission.addOracle(address(0));
        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](2);
        uint256 submitGas;
        uint256 batchSubmitGas;
        address[] memory aggregatorForBathSubmit = new address[](50);
        int256[] memory submissions = new int256[](50);
        uint256 startGas;

        for (uint i = 0; i < 50; i++) {
            Aggregator aggregator = new Aggregator(timeout, decimals, description);
            oracleAdd[0] = address(batchSubmission);
            oracleAdd[1] = address(0);
            aggregator.changeOracles(oracleRemove, oracleAdd, 1, 1, 0);
            aggregatorForBathSubmit[i] = address(aggregator);
            submissions[i] = 10;
            startGas = gasleft();
            vm.prank(address(0));
            aggregator.submit(10);
            submitGas += estimateGasCost(startGas);
        }
        startGas = gasleft();
        vm.prank(address(0));
        batchSubmission.batchSubmit(aggregatorForBathSubmit, submissions);
        batchSubmitGas = estimateGasCost(startGas);

        console.log("submit", submitGas, "batch submit", batchSubmitGas);
    }
}
