// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test, console} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";
import {IFeed} from "../src/interfaces/IFeed.sol";

contract SubmissionProxyTest is Test {
    SubmissionProxy submissionProxy;
    uint8 DECIMALS = 18;
    string[] SAMPLE_NAMES = [
        "BTC-USDT",
        "ETH-USDT",
        "XRP-USDT",
        "SOL-USDT",
        "ADA-USDT",
        "AVAX-USDT",
        "BNB-USDT",
        "DAI-USDT",
        "KLAT-USDT",
        "TRX-USDT",
        "UNI-USDT"
    ];
    uint256 timestamp = 1706170779;

    error OwnableUnauthorizedAccount(address account);

    function setUp() public {
        vm.warp(timestamp);
        submissionProxy = new SubmissionProxy();
    }

    function test_SetMaxSubmission() public {
        uint256 maxSubmission_ = (submissionProxy.MAX_SUBMISSION() - submissionProxy.MIN_SUBMISSION()) / 2;
        submissionProxy.setMaxSubmission(maxSubmission_);
        assertEq(submissionProxy.maxSubmission(), maxSubmission_);
    }

    function test_SetMaxSubmissionBelowMinimum() public {
        vm.expectRevert(SubmissionProxy.InvalidMaxSubmission.selector);
        // FAIL - 0 is not a valid max submission
        submissionProxy.setMaxSubmission(0);
    }

    function test_SetMaxSubmissionAboveMaximum() public {
        uint256 maxSubmission_ = submissionProxy.MAX_SUBMISSION();
        vm.expectRevert(SubmissionProxy.InvalidMaxSubmission.selector);
        // FAIL - MAX_SUBMISSION is the maximum allowed value
        submissionProxy.setMaxSubmission(maxSubmission_ + 1);
    }

    function test_SetMaxSubmissionProtectExecution() public {
        address nonOwner_ = makeAddr("non-owner");
        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        // FAIL - only owner can set max submission
        submissionProxy.setMaxSubmission(10);
    }

    function test_SetDataFreshness() public {
        uint256 dataFreshness_ = 1 days;
        submissionProxy.setDataFreshness(dataFreshness_);
        assertEq(submissionProxy.dataFreshness(), dataFreshness_);
    }

    function test_SetDataFreshnessProtextExecution() public {
        uint256 dataFreshness_ = 1 days;
        address nonOwner_ = makeAddr("non-owner");

        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        // FAIL - only owner can set data freshness
        submissionProxy.setDataFreshness(dataFreshness_);
    }

    function test_SetExpirationPeriod() public {
        uint256 expirationPeriod_ = (submissionProxy.MAX_EXPIRATION() - submissionProxy.MIN_EXPIRATION()) / 2;
        submissionProxy.setExpirationPeriod(expirationPeriod_);
        assertEq(submissionProxy.expirationPeriod(), expirationPeriod_);
    }

    function test_SetExpirationPeriodBelowMinimum() public {
        uint256 minExpiration_ = submissionProxy.MIN_EXPIRATION();
        vm.expectRevert(SubmissionProxy.InvalidExpirationPeriod.selector);
        // FAIL - expiration period must be greater than or equal to MIN_EXPIRATION
        submissionProxy.setExpirationPeriod(minExpiration_ / 2);
    }

    function test_SetExpirationPeriodAboveMaximum() public {
        uint256 maxExpiration_ = submissionProxy.MAX_EXPIRATION();
        vm.expectRevert(SubmissionProxy.InvalidExpirationPeriod.selector);
        // FAIL - expiration period must be less than or equal to MAX_EXPIRATION
        submissionProxy.setExpirationPeriod(maxExpiration_ + 1 days);
    }

    function test_SetExpirationPeriodProtectExecution() public {
        address nonOwner_ = makeAddr("non-owner");
        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        // FAIL - only owner can set expiration period
        submissionProxy.setExpirationPeriod(1 weeks);
    }

    function test_SetDefaultProofThreshold() public {
        uint8 threshold_ = submissionProxy.MIN_THRESHOLD();
        // SUCCESS - 1 is a valid threshold
        submissionProxy.setDefaultProofThreshold(threshold_);
        assertEq(submissionProxy.defaultThreshold(), threshold_);

        threshold_ = submissionProxy.MAX_THRESHOLD();
        // SUCCESS - 100 is a valid threshold
        submissionProxy.setDefaultProofThreshold(threshold_);
        assertEq(submissionProxy.defaultThreshold(), threshold_);

        threshold_ = submissionProxy.MIN_THRESHOLD() - 1;
        vm.expectRevert(SubmissionProxy.InvalidThreshold.selector);
        // FAIL - threshold must be greater than or equal to MIN_THRESHOLD
        submissionProxy.setDefaultProofThreshold(threshold_);

        threshold_ = submissionProxy.MAX_THRESHOLD() + 1;
        vm.expectRevert(SubmissionProxy.InvalidThreshold.selector);
        // FAIL - threshold must be less than or equal to MAX_THRESHOLD
        submissionProxy.setDefaultProofThreshold(threshold_);
    }

    function test_AddOracleProtectExecution() public {
        address oracle_ = makeAddr("oracle");
        address nonOwner_ = makeAddr("non-owner");

        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        // FAIL - only owner can add oracle
        submissionProxy.addOracle(oracle_);
    }

    function test_AddOracleOnce() public {
        address oracle_ = makeAddr("oracle");
        submissionProxy.addOracle(oracle_);
    }

    function test_AddOracleTwice() public {
        address oracle_ = makeAddr("oracle");

        submissionProxy.addOracle(oracle_);
        vm.expectRevert(SubmissionProxy.InvalidOracle.selector);
        // FAIL - cannot add the same oracle twice => fail
        submissionProxy.addOracle(oracle_);
    }

    function test_RemoveOracleProtectExecution() public {
        address oracle_ = makeAddr("oracle");
        address nonOwner_ = makeAddr("non-owner");

        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        // FAIL - only owner can remove oracle
        submissionProxy.removeOracle(oracle_);
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

    function test_UpdateOracleProtectExecution() public {
        address oracle_ = makeAddr("oracle");
        address nonOwner_ = makeAddr("non-owner");

        vm.prank(nonOwner_);
        vm.expectRevert(SubmissionProxy.OnlyOracle.selector);
        // FAIL - only owner can update oracle
        submissionProxy.updateOracle(oracle_);
    }

    function test_UpdateOracle() public {
        address oracle_ = makeAddr("oracle");
        submissionProxy.addOracle(oracle_);

        address newOracle_ = makeAddr("new-oracle");

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
        vm.prank(oracle_);
        vm.expectRevert(SubmissionProxy.OnlyOracle.selector);
        submissionProxy.updateOracle(makeAddr("anything"));
    }

    function test_SubmitWithInvalidSubmissionLength() public {
        (
            bytes32[] memory feedHashes1_,
            int256[] memory submissions1_,
            bytes[] memory proofs1_,
            uint256[] memory timestamps1_
        ) = createSubmitParameters(1);

        (
            bytes32[] memory feedHashes2_,
            int256[] memory submissions2_,
            bytes[] memory proofs2_,
            uint256[] memory timestamps2_
        ) = createSubmitParameters(2);

        // FAIL - invalid feeds length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feedHashes2_, submissions1_, timestamps1_, proofs1_);

        // FAIL - invalid submissions length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feedHashes1_, submissions2_, timestamps1_, proofs1_);

        // FAIL - invalid proofs length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feedHashes1_, submissions1_, timestamps1_, proofs2_);

        // FAIL - invalid timestamps length
        vm.expectRevert(SubmissionProxy.InvalidSubmissionLength.selector);
        submissionProxy.submit(feedHashes1_, submissions1_, timestamps2_, proofs1_);
    }

    function test_SubmitInvalidProof() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (, uint256 nonOracleSk_) = makeAddrAndKey("non-oracle");
        (address oracle_, uint256 oracleSk_) = makeAddrAndKey("oracle");
        submissionProxy.addOracle(oracle_);

        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_,
            address[] memory feeds_
        ) = prepareFeedsSubmissions(numOracles_, submissionValue_, oracleSk_);

        bytes32 hash_ = keccak256(abi.encodePacked(submissions_[0], timestamps_[0], feedHashes_[0]));

        // overwrite valid proof with a proof generated by non-oracle -> invalid proof
        proofs_[0] = abi.encodePacked(createProof(nonOracleSk_, hash_));

        // submission with invalid proof does not fail
        submissionProxy.submit(feedHashes_, submissions_, timestamps_, proofs_);

        // but no value is stored in the `Feed` contract
        vm.expectRevert(Feed.NoDataPresent.selector);
        IFeed(feeds_[0]).latestRoundData();
    }

    function test_SubmitStaleData() public {
        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (address oracle_, uint256 oracleSk_) = makeAddrAndKey("oracle");
        submissionProxy.addOracle(oracle_);

        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_,
            address[] memory feeds_
        ) = prepareFeedsSubmissions(numOracles_, submissionValue_, oracleSk_);

        uint256 doubleDataFreshnessPeriod_ = 2 * submissionProxy.dataFreshness();
        bytes32 hash_ =
            keccak256(abi.encodePacked(submissions_[0], block.timestamp + doubleDataFreshnessPeriod_, feedHashes_[0]));
        proofs_[0] = abi.encodePacked(createProof(oracleSk_, hash_));

        // submission with invalid data does not fail
        submissionProxy.submit(feedHashes_, submissions_, timestamps_, proofs_);

        // but no value is stored in the `Feed` contract
        vm.expectRevert(Feed.NoDataPresent.selector);
        IFeed(feeds_[0]).latestRoundData();
    }

    function test_SubmitIndexNotAscending() public {
        (address alice_, uint256 aliceSk_) = makeAddrAndKey("alice");
        (address bob_, uint256 bobSk_) = makeAddrAndKey("bob");
        (, uint256 dummySk_) = makeAddrAndKey("dummy");

        submissionProxy.addOracle(alice_); // index 0
        submissionProxy.addOracle(bob_); // index 1

        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_,
        ) = prepareFeedsSubmissions(numOracles_, submissionValue_, dummySk_);
        bytes32 hash_ = keccak256(abi.encodePacked(submissions_[0], timestamps_[0], feedHashes_[0]));

        // order of proofs is reversed!
        proofs_[0] = abi.encodePacked(createProof(bobSk_, hash_), createProof(aliceSk_, hash_));

        submissionProxy.setProofThreshold(feedHashes_[0], 100); // 100 % of the oracles must submit a valid proof

        vm.expectRevert(SubmissionProxy.IndexesNotAscending.selector);
        submissionProxy.submit(feedHashes_, submissions_, timestamps_, proofs_);
    }

    function test_SubmitUnregisteredFeed() public {
        (address alice_, uint256 aliceSk_) = makeAddrAndKey("alice");
        (address bob_, uint256 bobSk_) = makeAddrAndKey("bob");
        (address celine_, uint256 celineSk_) = makeAddrAndKey("celine");
        (, uint256 dummySk_) = makeAddrAndKey("dummy");

        submissionProxy.addOracle(alice_);
        submissionProxy.addOracle(bob_);
        submissionProxy.addOracle(celine_);

        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_,
            address[] memory feeds_
        ) = prepareFeedsSubmissions(numOracles_, submissionValue_, dummySk_);
        bytes32 hash_ = keccak256(abi.encodePacked(submissions_[0], timestamps_[0], feedHashes_[0]));
        proofs_[0] =
            abi.encodePacked(createProof(aliceSk_, hash_), createProof(bobSk_, hash_), createProof(celineSk_, hash_));

        submissionProxy.setProofThreshold(feedHashes_[0], 100); // 100 % of the oracles must submit a valid proof
        submissionProxy.removeFeed(feedHashes_[0]);
        submissionProxy.submit(feedHashes_, submissions_, timestamps_, proofs_);

        vm.expectRevert(Feed.NoDataPresent.selector);
        IFeed(feeds_[0]).latestRoundData();
    }

    function test_SubmitUnmatchedNameFeed() public {
        (address alice_, uint256 aliceSk_) = makeAddrAndKey("alice");
        (address bob_, uint256 bobSk_) = makeAddrAndKey("bob");
        (address celine_, uint256 celineSk_) = makeAddrAndKey("celine");
        (, uint256 dummySk_) = makeAddrAndKey("dummy");

        submissionProxy.addOracle(alice_);
        submissionProxy.addOracle(bob_);
        submissionProxy.addOracle(celine_);

        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_,
            address[] memory feeds_
        ) = prepareFeedsSubmissionsWrongName(numOracles_, submissionValue_, dummySk_);
        bytes32 hash_ = keccak256(abi.encodePacked(submissions_[0], timestamps_[0], feedHashes_[0]));
        proofs_[0] =
            abi.encodePacked(createProof(aliceSk_, hash_), createProof(bobSk_, hash_), createProof(celineSk_, hash_));

        submissionProxy.setProofThreshold(feedHashes_[0], 100); // 100 % of the oracles must submit a valid proof
        submissionProxy.submit(feedHashes_, submissions_, timestamps_, proofs_);

        vm.expectRevert(Feed.NoDataPresent.selector);
        IFeed(feeds_[0]).latestRoundData();
    }

    function test_SubmitCorrectProof() public {
        (address alice_, uint256 aliceSk_) = makeAddrAndKey("alice");
        (address bob_, uint256 bobSk_) = makeAddrAndKey("bob");
        (address celine_, uint256 celineSk_) = makeAddrAndKey("celine");
        (, uint256 dummySk_) = makeAddrAndKey("dummy");

        submissionProxy.addOracle(alice_);
        submissionProxy.addOracle(bob_);
        submissionProxy.addOracle(celine_);

        uint256 numOracles_ = 1;
        int256 submissionValue_ = 10;
        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_,
            address[] memory feeds_
        ) = prepareFeedsSubmissions(numOracles_, submissionValue_, dummySk_);
        bytes32 hash_ = keccak256(abi.encodePacked(submissions_[0], timestamps_[0], feedHashes_[0]));
        proofs_[0] =
            abi.encodePacked(createProof(aliceSk_, hash_), createProof(bobSk_, hash_), createProof(celineSk_, hash_));

        submissionProxy.setProofThreshold(feedHashes_[0], 100); // 100 % of the oracles must submit a valid proof
        submissionProxy.submit(feedHashes_, submissions_, timestamps_, proofs_);

        // don't raise `NoDataPresent`
        IFeed(feeds_[0]).latestRoundData();
    }

    function prepareFeedsSubmissions(uint256 _numOracles, int256 _submissionValue, uint256 _oracleSk)
        private
        returns (bytes32[] memory, int256[] memory, bytes[] memory, uint256[] memory, address[] memory)
    {
        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_
        ) = createSubmitParameters(_numOracles);
        address[] memory feeds_ = new address[](_numOracles);
        for (uint256 i = 0; i < _numOracles; i++) {
            Feed feed_ = new Feed(DECIMALS, SAMPLE_NAMES[i], address(submissionProxy));
            feeds_[i] = address(feed_);
            feedHashes_[i] = keccak256(abi.encodePacked(SAMPLE_NAMES[i]));
            submissions_[i] = _submissionValue;
            timestamps_[i] = block.timestamp * 1000;
            proofs_[i] =
                createProof(_oracleSk, keccak256(abi.encodePacked(timestamps_[i], submissions_[i], feedHashes_[i])));
            submissionProxy.updateFeed(feedHashes_[i], address(feeds_[i]));
        }

        return (feedHashes_, submissions_, proofs_, timestamps_, feeds_);
    }

    function prepareFeedsSubmissionsWrongName(uint256 _numOracles, int256 _submissionValue, uint256 _oracleSk)
        private
        returns (bytes32[] memory, int256[] memory, bytes[] memory, uint256[] memory, address[] memory)
    {
        (
            bytes32[] memory feedHashes_,
            int256[] memory submissions_,
            bytes[] memory proofs_,
            uint256[] memory timestamps_
        ) = createSubmitParameters(_numOracles);
        address[] memory feeds_ = new address[](_numOracles);
        for (uint256 i = 0; i < _numOracles; i++) {
            Feed feed_ = new Feed(DECIMALS, "wrong-name", address(submissionProxy));
            feeds_[i] = address(feed_);
            feedHashes_[i] = keccak256(abi.encodePacked(SAMPLE_NAMES[i]));
            submissions_[i] = _submissionValue;
            timestamps_[i] = block.timestamp * 1000;
            proofs_[i] =
                createProof(_oracleSk, keccak256(abi.encodePacked(timestamps_[i], submissions_[i], feedHashes_[i])));
            submissionProxy.updateFeed(feedHashes_[i], address(feeds_[i]));
        }

        return (feedHashes_, submissions_, proofs_, timestamps_, feeds_);
    }

    function createSubmitParameters(uint256 numSubmissions)
        private
        pure
        returns (bytes32[] memory, int256[] memory, bytes[] memory, uint256[] memory)
    {
        bytes32[] memory feedHashes_ = new bytes32[](numSubmissions);
        int256[] memory submissions_ = new int256[](numSubmissions);
        bytes[] memory proofs_ = new bytes[](numSubmissions);
        uint256[] memory timestamps_ = new uint256[](numSubmissions);

        return (feedHashes_, submissions_, proofs_, timestamps_);
    }

    function createProof(uint256 sk, bytes32 hash) internal pure returns (bytes memory) {
        (uint8 v_, bytes32 r_, bytes32 s_) = vm.sign(sk, hash);
        return abi.encodePacked(r_, s_, v_);
    }

    function estimateGasCost(uint256 _startGas) internal view returns (uint256) {
        return _startGas - gasleft();
    }

    function test_ProofGerationAndRecovery() public {
        // off-chain
        (address oracle_, uint256 sk_) = makeAddrAndKey("oracle");
        uint256 ts_ = block.timestamp;
        int256 answer_ = 10;
        bytes32 hash_ = keccak256(abi.encodePacked(ts_, answer_));
        (uint8 v_, bytes32 r_, bytes32 s_) = vm.sign(sk_, hash_);

        // on-chain
        address recoveredOracle_ = ecrecover(hash_, v_, r_, s_);
        assertEq(recoveredOracle_, oracle_);
    }
}
