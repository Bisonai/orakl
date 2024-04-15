// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";
import {IFeed} from "../src/interfaces/IFeed.sol";

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

    function test_submitInvalidThreshold() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");

        (address[] memory feeds_, int256[] memory submissions_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_);

	bytes[] memory proofs_ = new bytes[](1);
	proofs_[0] = "invalid-proof";

        vm.expectRevert(SubmissionProxy.InvalidThreshold.selector);
        submissionProxy.submit(feeds_, submissions_, proofs_);
    }

    function test_submitInvalidProof() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");

        (address[] memory feeds_, int256[] memory submissions_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_);

	bytes[] memory proofs_ = new bytes[](1);
	proofs_[0] = "invalid-proof";

	submissionProxy.setProofThreshold(feeds_[0], 1);

        submissionProxy.submit(feeds_, submissions_, proofs_);

        vm.expectRevert(Feed.NoDataPresent.selector);
	IFeed(feeds_[0]).latestRoundData();
    }

    function prepareFeedsSubmissions(uint256 _numOracles, int256 _submissionValue, address _oracle)
        internal
        returns (address[] memory, int256[] memory)
    {
        submissionProxy.addOracle(_oracle);

        address[] memory feeds_ = new address[](_numOracles);
        int256[] memory submissions_ = new int256[](_numOracles);

        for (uint256 i = 0; i < _numOracles; i++) {
            Feed feed_ = new Feed(DECIMALS, DESCRIPTION, address(submissionProxy));

            feeds_[i] = address(feed_);
            submissions_[i] = _submissionValue;
        }

        return (feeds_, submissions_);
    }

    function test_submitWithProof() public {
        address offChainSubmissionProxyReporter = makeAddr("submission-proxy-reporter");
        submissionProxy.addOracle(offChainSubmissionProxyReporter);
        (, uint256 aliceSk) = makeAddrAndKey("alice");
        (, uint256 bobSk) = makeAddrAndKey("bob");
        (, uint256 celineSk) = makeAddrAndKey("celine");

        bytes32 hash = keccak256(abi.encodePacked(int256(10)));

        uint256 numSubmissions = 1;
        address[] memory feeds = new address[](numSubmissions);
        int256[] memory submissions = new int256[](numSubmissions);
        bytes[] memory proofs = new bytes[](numSubmissions);

        // single data feed
        Feed feed = new Feed(DECIMALS, DESCRIPTION, address(submissionProxy));

        feeds[0] = address(feed);
        submissions[0] = 10;

	submissionProxy.setProofThreshold(feeds[0], 3);

        /* proofs[0] = abi.encodePacked(createProof(aliceSk, hash)); */
        /* proofs[0] = abi.encodePacked(createProof(aliceSk, hash), createProof(bobSk, hash)); */
        proofs[0] = abi.encodePacked(createProof(aliceSk, hash), createProof(bobSk, hash), createProof(celineSk, hash));

        submitWithProof(offChainSubmissionProxyReporter, feeds, submissions, proofs); // warmup
        submitWithProof(offChainSubmissionProxyReporter, feeds, submissions, proofs);
    }

    function submitWithProof(
        address reporter,
        address[] memory feeds,
        int256[] memory submissions,
        bytes[] memory proofs
    ) internal {
        vm.prank(reporter);
        uint256 startGas = gasleft();
        submissionProxy.submit(feeds, submissions, proofs);
        uint256 gas = estimateGasCost(startGas);
        console.log("w/ proof", gas);
    }

    function createProof(uint256 sk, bytes32 hash) internal pure returns (bytes memory) {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, hash);
        return abi.encodePacked(r, s, v);
    }
}
