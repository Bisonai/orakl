// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console2, console} from "forge-std/Test.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Aggregator} from "../src/Aggregator.sol";

// TODO test submit after oracle expires
contract SubmissionProxyTest is Test {
    SubmissionProxy submissionProxy;
    uint32 TIMEOUT = 10;
    uint8 DECIMALS = 18;
    string DESCRIPTION = "Test Aggregator";
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
        uint256 numOracles_ = submissionProxy.getOracles().length;
        assertEq(numOracles_, 1);

        submissionProxy.removeOracle(address(0));
        numOracles_ = submissionProxy.getOracles().length;
        assertEq(numOracles_, 0);
    }

    function test_GetExpiredOracles() public {
        address oracle_ = makeAddr("oracle");

        submissionProxy.addOracle(oracle_);
        uint256 numOracles_ = submissionProxy.getOracles().length;
        assertEq(numOracles_, 1);

        // oracle has not expired yet
        address[] memory expired_ = submissionProxy.getExpiredOracles();
        assertEq(expired_.length, 0);

        // oracle expired
        vm.warp(block.timestamp + submissionProxy.expirationPeriod() + 1);
        expired_ = submissionProxy.getExpiredOracles();
        assertEq(expired_.length, 1);
        assertEq(expired_[0], oracle_);
    }

    function test_SetMaxSubmission() public {
        uint256 maxSubmission_ = 10;
        submissionProxy.setMaxSubmission(maxSubmission_);
        assertEq(submissionProxy.maxSubmission(), maxSubmission_);
    }

    function testFail_SetMaxSubmissionProtectExecution() public {
        address nonOwner_ = makeAddr("nonOwner");
        vm.prank(nonOwner_);
        submissionProxy.setMaxSubmission(10);
    }

    function test_SetExpirationPeriod() public {
        uint256 expirationPeriod_ = 1 weeks;
        submissionProxy.setExpirationPeriod(expirationPeriod_);
        assertEq(submissionProxy.expirationPeriod(), expirationPeriod_);
    }

    function testFail_SetExpirationPeriodProtectExecution() public {
        address nonOwner_ = makeAddr("nonOwner");
        vm.prank(nonOwner_);
        submissionProxy.setExpirationPeriod(1 weeks);
    }

    function prepareAggregatorsSubmissions(uint256 _numOracles, int256 _submissionValue, address _oracle)
        internal
        returns (address[] memory, int256[] memory)
    {
        submissionProxy.addOracle(_oracle);

        address[] memory remove_;
        address[] memory add_ = new address[](2);
        add_[0] = address(submissionProxy);
        add_[1] = _oracle;

        address[] memory aggregators_ = new address[](_numOracles);
        int256[] memory submissions_ = new int256[](_numOracles);

        for (uint256 i = 0; i < _numOracles; i++) {
            Aggregator aggregator_ = new Aggregator(TIMEOUT, DECIMALS, DESCRIPTION);

            aggregator_.changeOracles(remove_, add_, 1, 1, 0);

            aggregators_[i] = address(aggregator_);
            submissions_[i] = _submissionValue;
        }

        return (aggregators_, submissions_);
    }

    function testFail_submitWithExpiredOracle() public {
        uint256 numOracles_ = 2;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");

        (address[] memory aggregators_, int256[] memory submissions_) =
            prepareAggregatorsSubmissions(numOracles_, submissionValue_, oracle_);

        // move time past the expiration period => fail to submit
        vm.warp(block.timestamp + submissionProxy.expirationPeriod() + 1);

        vm.prank(oracle_);
        submissionProxy.submit(aggregators_, submissions_);
    }

    function testFail_submitWithNonOracle() public {
        uint256 numOracles_ = 2;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");
        address nonOracle_ = makeAddr("nonOracle");

        (address[] memory aggregators_, int256[] memory submissions_) =
            prepareAggregatorsSubmissions(numOracles_, submissionValue_, oracle_);

        // only oracle can submit through submission proxy => fail to submit
        vm.prank(nonOracle_);
        submissionProxy.submit(aggregators_, submissions_);
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
            Aggregator aggregator = new Aggregator(TIMEOUT, DECIMALS, DESCRIPTION);

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
