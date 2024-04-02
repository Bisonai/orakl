// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";

contract SubmissionProxyTest is Test {
    SubmissionProxy submissionProxy;
    uint8 DECIMALS = 18;
    string DESCRIPTION = "Test Feed";
    uint256 timestamp = 1706170779;

    function estimateGasCost(uint256 startGas) internal view returns (uint256) {
        return startGas - gasleft();
    }

    function setUp() public {
        vm.warp(timestamp);
        submissionProxy = new SubmissionProxy();
    }

    function test_AddOracleOnce() public {
        address oracle_ = makeAddr("oracle");
        submissionProxy.addOracle(oracle_);
    }

    function test_AddOracleTwice() public {
        address oracle_ = makeAddr("oracle");
        submissionProxy.addOracle(oracle_);

        // cannot add the same oracle twice => fail
        vm.expectRevert(SubmissionProxy.InvalidOracle.selector);
        submissionProxy.addOracle(oracle_);
    }

    function test_SetMaxSubmission() public {
        uint256 maxSubmission_ = 10;
        submissionProxy.setMaxSubmission(maxSubmission_);
        assertEq(submissionProxy.maxSubmission(), maxSubmission_);
    }

    function test_SetMaxSubmissionBelowMinimum() public {
        vm.expectRevert(SubmissionProxy.InvalidMaxSubmission.selector);
        submissionProxy.setMaxSubmission(0);
    }

    function test_SetMaxSubmissionAboveMaximum() public {
        uint256 maxSubmission_ = submissionProxy.MAX_SUBMISSION();
        vm.expectRevert(SubmissionProxy.InvalidMaxSubmission.selector);
        submissionProxy.setMaxSubmission(maxSubmission_ + 1);
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

    function test_SetExpirationPeriodBelowMinimum() public {
        uint256 minExpiration_ = submissionProxy.MIN_EXPIRATION();
        vm.expectRevert(SubmissionProxy.InvalidExpirationPeriod.selector);
        submissionProxy.setExpirationPeriod(minExpiration_ / 2);
    }

    function test_SetExpirationPeriodAboveMaximum() public {
        uint256 maxExpiration_ = submissionProxy.MAX_EXPIRATION();
        vm.expectRevert(SubmissionProxy.InvalidExpirationPeriod.selector);
        submissionProxy.setExpirationPeriod(maxExpiration_ + 1 days);
    }

    function testFail_SetExpirationPeriodProtectExecution() public {
        address nonOwner_ = makeAddr("nonOwner");
        vm.prank(nonOwner_);
        submissionProxy.setExpirationPeriod(1 weeks);
    }

    function testFail_submitWithExpiredOracle() public {
        uint256 numOracles_ = 2;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");

        (address[] memory feeds_, int256[] memory submissions_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_);

        // move time past the expiration period => fail to submit
        vm.warp(block.timestamp + submissionProxy.expirationPeriod() + 1);

        vm.prank(oracle_);
        submissionProxy.submit(feeds_, submissions_);
    }

    function testFail_submitWithNonOracle() public {
        uint256 numOracles_ = 2;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");
        address nonOracle_ = makeAddr("nonOracle");

        (address[] memory feeds_, int256[] memory submissions_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_);

        // only oracle can submit through submission proxy => fail to submit
        vm.prank(nonOracle_);
        submissionProxy.submit(feeds_, submissions_);
    }

    function test_BatchSubmission() public {
        uint256 numOracles = 50;
        address offChainSubmissionProxyReporter = makeAddr("submission-proxy-reporter");
        address offChainFeedReporter = makeAddr("off-chain-reporter");

        submissionProxy.addOracle(offChainSubmissionProxyReporter);

        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](2);
        uint256 singleSubmissionGas;
        uint256 batchSubmissionGas;
        address[] memory feeds = new address[](numOracles);
        int256[] memory submissions = new int256[](numOracles);
        uint256 startGas;

        // multiple single submissions
        for (uint256 i = 0; i < numOracles; i++) {
            Feed feed = new Feed(DECIMALS, DESCRIPTION);
	    feed.setProofRequired(false);

            oracleAdd[0] = address(submissionProxy);
            oracleAdd[1] = offChainFeedReporter;
            feed.changeOracles(oracleRemove, oracleAdd);

            feeds[i] = address(feed);
            submissions[i] = 10;

            vm.prank(offChainFeedReporter);
            feed.submit(10); // storage warmup

            vm.prank(offChainFeedReporter);
            startGas = gasleft();
            feed.submit(11);
            singleSubmissionGas += estimateGasCost(startGas);
        }

        // single batch submission
        vm.prank(offChainSubmissionProxyReporter);
        submissionProxy.submit(feeds, submissions); // storage warmup

        vm.prank(offChainSubmissionProxyReporter);
        startGas = gasleft();
        submissionProxy.submit(feeds, submissions);
        batchSubmissionGas = estimateGasCost(startGas);

        console.log("single submit", singleSubmissionGas, "batch submit", batchSubmissionGas);
        if (singleSubmissionGas > batchSubmissionGas) {
            console.log("save", singleSubmissionGas - batchSubmissionGas);
        } else {
            console.log("waste", batchSubmissionGas - singleSubmissionGas);
        }
    }

    function prepareFeedsSubmissions(uint256 _numOracles, int256 _submissionValue, address _oracle)
        internal
        returns (address[] memory, int256[] memory)
    {
        submissionProxy.addOracle(_oracle);

        address[] memory remove_;
        address[] memory add_ = new address[](2);
        add_[0] = address(submissionProxy);
        add_[1] = _oracle;

        address[] memory feeds_ = new address[](_numOracles);
        int256[] memory submissions_ = new int256[](_numOracles);

        for (uint256 i = 0; i < _numOracles; i++) {
            Feed feed_ = new Feed(DECIMALS, DESCRIPTION);

            feed_.changeOracles(remove_, add_);

            feeds_[i] = address(feed_);
            submissions_[i] = _submissionValue;
        }

        return (feeds_, submissions_);
    }

    function test_submitWithProof() public {
	address offChainSubmissionProxyReporter = makeAddr("submission-proxy-reporter");
	submissionProxy.addOracle(offChainSubmissionProxyReporter);
	(address alice, uint256 aliceSk) = makeAddrAndKey("alice");
	(address bob, uint256 bobSk) = makeAddrAndKey("bob");
	(address celine, uint256 celineSk) = makeAddrAndKey("celine");

	bytes32 hash = keccak256(abi.encodePacked(int256(10)));

	uint256 numSubmissions = 1;
	address[] memory feeds = new address[](numSubmissions);
	int256[] memory submissions = new int256[](numSubmissions);
	bytes[] memory proofs = new bytes[](numSubmissions);

	// single data feed
        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](4);
	Feed feed = new Feed(DECIMALS, DESCRIPTION);
	feed.setProofRequired(false);
	oracleAdd[0] = address(submissionProxy);
	oracleAdd[1] = address(alice);
	oracleAdd[2] = address(bob);
	oracleAdd[3] = address(celine);
	feed.changeOracles(oracleRemove, oracleAdd);

	feeds[0] = address(feed);
	submissions[0] = 10;

	/* proofs[0] = abi.encodePacked(createProof(aliceSk, hash)); */
	/* proofs[0] = abi.encodePacked(createProof(aliceSk, hash), createProof(bobSk, hash)); */
	proofs[0] = abi.encodePacked(createProof(aliceSk, hash), createProof(bobSk, hash), createProof(celineSk, hash));

	submitWithProof(offChainSubmissionProxyReporter, feeds, submissions, proofs); // warmup
	submitWithProof(offChainSubmissionProxyReporter, feeds, submissions, proofs);

	submitWithoutProof(offChainSubmissionProxyReporter, feeds, submissions); // warmup
	submitWithoutProof(offChainSubmissionProxyReporter, feeds, submissions);
    }

    function submitWithProof(address reporter, address[] memory feeds, int256[] memory submissions, bytes[] memory proofs) internal {
	vm.prank(reporter);
	uint256 startGas = gasleft();
	submissionProxy.submit(feeds, submissions, proofs);
	uint256 gas = estimateGasCost(startGas);
	console.log("w/ proof", gas);
    }

    function submitWithoutProof(address reporter, address[] memory feeds, int256[] memory submissions) internal {
	vm.prank(reporter);
	uint256 startGas = gasleft();
	submissionProxy.submit(feeds, submissions);
	uint256 gas = estimateGasCost(startGas);
	console.log("w/o proof", gas);
    }

    function createProof(uint256 sk, bytes32 hash) internal pure returns (bytes memory) {
	(uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, hash);
	return abi.encodePacked(r, s, v);
    }
}
