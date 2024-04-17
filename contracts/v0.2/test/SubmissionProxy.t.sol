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
        {
            submissionProxy.addOracle(oracle_);
            assertEq(submissionProxy.oracles(0), oracle_);
            (, uint256 expirationTime_) = submissionProxy.whitelist(oracle_);
            assertGe(expirationTime_, block.timestamp);
        }

        // remove
        {
            submissionProxy.removeOracle(oracle_);

            vm.expectRevert(bytes("")); // "EvmError: Revert"
            submissionProxy.oracles(0);

            (, uint256 expirationTime_) = submissionProxy.whitelist(oracle_);
            assertEq(expirationTime_, block.timestamp);
        }
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
        uint256 duringUpdateTimestamp_ = block.timestamp;
        {
            (, uint256 expirationTime_) = submissionProxy.whitelist(oracle_);
            assertGe(expirationTime_, duringUpdateTimestamp_);
        }

        // SUCCESS - Registered oracle can update its address
        vm.prank(oracle_);
        submissionProxy.updateOracle(newOracle_);

        // Old oracle has expiration time changed to the timestamp
        // during which the update occured.
        {
            (, uint256 expirationTime_) = submissionProxy.whitelist(oracle_);
            assertEq(expirationTime_, duringUpdateTimestamp_);
        }

        // New oracle has expiration time larger than the current block timestamp.
        {
            (, uint256 expirationTime_) = submissionProxy.whitelist(newOracle_);
            assertGe(expirationTime_, block.timestamp);
        }

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

    function test_SubmitWithInvalidSubmissionLength() public {
        (
            address[] memory feeds1_,
            int256[] memory submissions1_,
            bytes[] memory proofs1_,
            uint256[] memory timestamps1_
        ) = createSubmitParameters(1);

        (
            address[] memory feeds2_,
            int256[] memory submissions2_,
            bytes[] memory proofs2_,
            uint256[] memory timestamps2_
        ) = createSubmitParameters(2);

        // FAIL - invalid feeds length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feeds2_, submissions1_, proofs1_, timestamps1_);

        // FAIL - invalid submissions length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feeds1_, submissions2_, proofs1_, timestamps1_);

        // FAIL - invalid proofs length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feeds1_, submissions1_, proofs2_, timestamps1_);

        // FAIL - invalid timestamps length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feeds1_, submissions1_, proofs1_, timestamps2_);
    }

    function test_SubmitInvalidProof() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (address oracle_, uint256 oracleSk_) = makeAddrAndKey("oracle");
        (, uint256 nonOracleSk_) = makeAddrAndKey("non-oracle");

        (address[] memory feeds_, int256[] memory submissions_, bytes[] memory proofs_, uint256[] memory timestamps_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_, oracleSk_);

        bytes32 hash_ = keccak256(abi.encodePacked(timestamps_[0], int256(submissions_[0])));

        // overwrite valid proof with a proof generated by non-oracle -> invalid proof
        proofs_[0] = abi.encodePacked(createProof(nonOracleSk_, hash_));

        // submission with invalid proof does not fail
        submissionProxy.submit(feeds_, submissions_, proofs_, timestamps_);

        // but no value is stored in the `Feed` contract
        vm.expectRevert(Feed.NoDataPresent.selector);
        IFeed(feeds_[0]).latestRoundData();
    }

    function test_SubmitStaleData() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (address oracle_, uint256 oracleSk_) = makeAddrAndKey("oracle");

        (address[] memory feeds_, int256[] memory submissions_, bytes[] memory proofs_, uint256[] memory timestamps_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, oracle_, oracleSk_);

        uint256 doubleDataFreshnessPeriod_ = 2 * submissionProxy.dataFreshness();
        bytes32 hash_ =
            keccak256(abi.encodePacked(block.timestamp + doubleDataFreshnessPeriod_, int256(submissions_[0])));
        proofs_[0] = abi.encodePacked(createProof(oracleSk_, hash_));

        // submission with invalid data does not fail
        submissionProxy.submit(feeds_, submissions_, proofs_, timestamps_);

        // but no value is stored in the `Feed` contract
        vm.expectRevert(Feed.NoDataPresent.selector);
        IFeed(feeds_[0]).latestRoundData();
    }

    function test_SubmitWithProof() public {
        (address alice_, uint256 aliceSk_) = makeAddrAndKey("alice");
        (address bob_, uint256 bobSk_) = makeAddrAndKey("bob");
        (address celine_, uint256 celineSk_) = makeAddrAndKey("celine");
        (address dummy_, uint256 dummySk_) = makeAddrAndKey("dummy");

        submissionProxy.addOracle(alice_);
        submissionProxy.addOracle(bob_);
        submissionProxy.addOracle(celine_);

        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (address[] memory feeds_, int256[] memory submissions_, bytes[] memory proofs_, uint256[] memory timestamps_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, dummy_, dummySk_);
        bytes32 hash_ = keccak256(abi.encodePacked(timestamps_[0], int256(submissions_[0])));
        proofs_[0] =
            abi.encodePacked(createProof(aliceSk_, hash_), createProof(bobSk_, hash_), createProof(celineSk_, hash_));

        submissionProxy.setProofThreshold(feeds_[0], 100); // 100 % of the oracles must submit a valid proof
        submissionProxy.submit(feeds_, submissions_, proofs_, timestamps_);
    }

    function test_SubmitIndexNotAscending() public {
        (address alice_, uint256 aliceSk_) = makeAddrAndKey("alice");
        (address bob_, uint256 bobSk_) = makeAddrAndKey("bob");
        (address dummy_, uint256 dummySk_) = makeAddrAndKey("dummy");

        submissionProxy.addOracle(alice_); // index 0
        submissionProxy.addOracle(bob_); // index 1

        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (address[] memory feeds_, int256[] memory submissions_, bytes[] memory proofs_, uint256[] memory timestamps_) =
            prepareFeedsSubmissions(numOracles_, submissionValue_, dummy_, dummySk_);
        bytes32 hash_ = keccak256(abi.encodePacked(timestamps_[0], submissions_[0]));

        // order of proofs is reversed!
        proofs_[0] = abi.encodePacked(createProof(bobSk_, hash_), createProof(aliceSk_, hash_));

        vm.expectRevert(SubmissionProxy.IndexesNotAscending.selector);
        submissionProxy.submit(feeds_, submissions_, proofs_, timestamps_);
    }

    function prepareFeedsSubmissions(uint256 _numOracles, int256 _submissionValue, address _oracle, uint256 _oracleSk)
        private
        returns (address[] memory, int256[] memory, bytes[] memory, uint256[] memory)
    {
        submissionProxy.addOracle(_oracle);

        (address[] memory feeds_, int256[] memory submissions_, bytes[] memory proofs_, uint256[] memory timestamps_) =
            createSubmitParameters(_numOracles);

        for (uint256 i = 0; i < _numOracles; i++) {
            Feed feed_ = new Feed(DECIMALS, DESCRIPTION, address(submissionProxy));

            feeds_[i] = address(feed_);
            submissions_[i] = _submissionValue;
            timestamps_[i] = block.timestamp;
            proofs_[i] = createProof(_oracleSk, keccak256(abi.encodePacked(timestamps_[i], submissions_[i])));
        }

        return (feeds_, submissions_, proofs_, timestamps_);
    }

    function createSubmitParameters(uint256 numSubmissions)
        private
        pure
        returns (address[] memory, int256[] memory, bytes[] memory, uint256[] memory)
    {
        address[] memory feeds_ = new address[](numSubmissions);
        int256[] memory submissions_ = new int256[](numSubmissions);
        bytes[] memory proofs_ = new bytes[](numSubmissions);
        uint256[] memory timestamps_ = new uint256[](numSubmissions);

        return (feeds_, submissions_, proofs_, timestamps_);
    }

    function createProof(uint256 sk, bytes32 hash) internal pure returns (bytes memory) {
        (uint8 v_, bytes32 r_, bytes32 s_) = vm.sign(sk, hash);
        return abi.encodePacked(r_, s_, v_);
    }

    function estimateGasCost(uint256 _startGas) internal view returns (uint256) {
        return _startGas - gasleft();
    }
}
