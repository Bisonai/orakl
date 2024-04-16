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

    function test_RemoveOracle() public {
        address oracle_ = makeAddr("oracle");

        // add
        submissionProxy.addOracle(oracle_);
        assertEq(submissionProxy.oracles(0), oracle_);
        assertGe(submissionProxy.whitelist(oracle_), block.timestamp);

        // remove
        submissionProxy.removeOracle(oracle_);
        vm.expectRevert(bytes("")); // "EvmError: Revert"
        submissionProxy.oracles(0);
        assertEq(submissionProxy.whitelist(oracle_), block.timestamp);
    }

    function test_UpdateOracle() public {
        address oracle_ = makeAddr("oracle");
        submissionProxy.addOracle(oracle_);

        address newOracle_ = makeAddr("new-oracle");
        address nonOracle_ = makeAddr("non-oracle");

        // FAIL - Only registered oracle can update its address
        vm.prank(nonOracle_);
        vm.expectRevert(SubmissionProxy.OnlyOracle.selector);
        submissionProxy.updateOracle(newOracle_);

        // Expiration time is larger than the current block timestamp.
        uint256 duringUpdateTimestamp = block.timestamp;
        assertGe(submissionProxy.whitelist(oracle_), duringUpdateTimestamp);

        // SUCCESS - Registered oracle can update its address
        vm.prank(oracle_);
        submissionProxy.updateOracle(newOracle_);

        // Old oracle has expiration time changed to the timestamp
        // during which the update occured.
        assertEq(submissionProxy.whitelist(oracle_), duringUpdateTimestamp);

        // New oracle has expiration time larger than the current block timestamp.
        assertGe(submissionProxy.whitelist(newOracle_), block.timestamp);

        // FAIL - Cannot update with outdated oracle address
        address newestOracle_ = makeAddr("newest-oracle");
        vm.expectRevert(SubmissionProxy.OnlyOracle.selector);
        vm.prank(oracle_);
        submissionProxy.updateOracle(newestOracle_);
    }

    function test_SetDefaultProofThreshold() public {
        // SUCCESS - 1 is a valid threshold
        uint8 defaultProofThreshold_ = 1;
        submissionProxy.setDefaultProofThreshold(defaultProofThreshold_);
        assertEq(submissionProxy.threshold(), defaultProofThreshold_);

        // SUCCESS - 100 is a valid threshold
        defaultProofThreshold_ = 100;
        submissionProxy.setDefaultProofThreshold(defaultProofThreshold_);
        assertEq(submissionProxy.threshold(), defaultProofThreshold_);

        // FAIL - 0 is not a valid threshold
        defaultProofThreshold_ = 0;
        vm.expectRevert(SubmissionProxy.InvalidThreshold.selector);
        submissionProxy.setDefaultProofThreshold(defaultProofThreshold_);

        // FAIL - 101 is not a valid threshold
        defaultProofThreshold_ = 101;
        vm.expectRevert(SubmissionProxy.InvalidThreshold.selector);
        submissionProxy.setDefaultProofThreshold(defaultProofThreshold_);
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

    function test_SubmitInvalidProof() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        address oracle_ = makeAddr("oracle");

        (address[] memory feeds_, int256[] memory submissions_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_);

        bytes[] memory proofs_ = new bytes[](1);
        proofs_[0] = "invalid-proof";

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

    function test_SubmitWithProof() public {
        (address alice, uint256 aliceSk) = makeAddrAndKey("alice");
        (address bob, uint256 bobSk) = makeAddrAndKey("bob");
        (address celine, uint256 celineSk) = makeAddrAndKey("celine");

        uint256 aliceIdx = submissionProxy.addOracle(alice);
        uint256 bobIdx = submissionProxy.addOracle(bob);
        uint256 celineIdx = submissionProxy.addOracle(celine);

        bytes32 hash = keccak256(abi.encodePacked(int256(10)));

        uint256 numSubmissions = 1;
        (address[] memory feeds, int256[] memory submissions, bytes[] memory proofs) =
            createSubmitParameters(numSubmissions);

        // single data feed
        Feed feed = new Feed(DECIMALS, DESCRIPTION, address(submissionProxy));

        feeds[0] = address(feed);
        submissions[0] = 10;

        submissionProxy.setProofThreshold(feeds[0], 100);

        /* proofs[0] = abi.encodePacked(uint8(aliceIdx), createProof(aliceSk, hash)); */
        /* proofs[0] = abi.encodePacked(uint8(aliceIdx), createProof(aliceSk, hash), uint8(bobIdx), createProof(bobSk, hash)); */
        proofs[0] = abi.encodePacked(
            uint8(aliceIdx),
            createProof(aliceSk, hash),
            uint8(bobIdx),
            createProof(bobSk, hash),
            uint8(celineIdx),
            createProof(celineSk, hash)
        );

        submissionProxy.submit(feeds, submissions, proofs);
    }

    function test_SubmitIndexOutOfBounds() public {
        (address alice, uint256 aliceSk) = makeAddrAndKey("alice");
        (address bob, uint256 bobSk) = makeAddrAndKey("bob");

        uint256 aliceIdx = submissionProxy.addOracle(alice); // index 0
        uint256 bobIdx = submissionProxy.addOracle(bob); // index 1
        uint256 wrongIdx = 3; // cannot be index 3 (out of bounds)
        assertEq((wrongIdx - bobIdx) != 0, true);

        uint256 numSubmissions = 1;
        (address[] memory feeds, int256[] memory submissions, bytes[] memory proofs) =
            createSubmitParameters(numSubmissions);

        Feed feed = new Feed(DECIMALS, DESCRIPTION, address(submissionProxy));

        feeds[0] = address(feed);
        submissions[0] = 10;
        bytes32 hash = keccak256(abi.encodePacked(int256(submissions[0])));
        proofs[0] =
            abi.encodePacked(uint8(aliceIdx), createProof(aliceSk, hash), uint8(wrongIdx), createProof(bobSk, hash));

        vm.expectRevert(SubmissionProxy.IndexOutOfBounds.selector);
        submissionProxy.submit(feeds, submissions, proofs);
    }

    function test_SubmitIndexNotAscending() public {
        (address alice, uint256 aliceSk) = makeAddrAndKey("alice");
        (address bob, uint256 bobSk) = makeAddrAndKey("bob");

        uint256 aliceIdx = submissionProxy.addOracle(alice); // index 0
        uint256 bobIdx = submissionProxy.addOracle(bob); // index 1
        uint256 wrongIdx = 0; // cannot be index 0 (repetitive index)
        assertGe(bobIdx, wrongIdx);

        uint256 numSubmissions = 1;
        (address[] memory feeds, int256[] memory submissions, bytes[] memory proofs) =
            createSubmitParameters(numSubmissions);

        Feed feed = new Feed(DECIMALS, DESCRIPTION, address(submissionProxy));

        feeds[0] = address(feed);
        submissions[0] = 10;
        bytes32 hash = keccak256(abi.encodePacked(int256(submissions[0])));
        proofs[0] =
            abi.encodePacked(uint8(aliceIdx), createProof(aliceSk, hash), uint8(wrongIdx), createProof(bobSk, hash));

        vm.expectRevert(SubmissionProxy.IndexesNotAscending.selector);
        submissionProxy.submit(feeds, submissions, proofs);
    }

    function createSubmitParameters(uint256 numSubmissions)
        internal
        pure
        returns (address[] memory, int256[] memory, bytes[] memory)
    {
        address[] memory feeds = new address[](numSubmissions);
        int256[] memory submissions = new int256[](numSubmissions);
        bytes[] memory proofs = new bytes[](numSubmissions);

        return (feeds, submissions, proofs);
    }

    function createProof(uint256 sk, bytes32 hash) internal pure returns (bytes memory) {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, hash);
        return abi.encodePacked(r, s, v);
    }
}
