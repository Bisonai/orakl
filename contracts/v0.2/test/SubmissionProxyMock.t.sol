// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test, console} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";
import {IFeed} from "../src/interfaces/IFeed.sol";

contract SubmissionProxyMock is SubmissionProxy {
    function callQuorum(uint8 _threshold) external view returns (uint256) {
        return quorum(_threshold);
    }

    function callSplitSignature(bytes memory _sig) external pure returns (uint8, bytes32, bytes32) {
        return splitSignature(_sig);
    }

    function callSplitProofs(bytes memory _data) external pure returns (bytes[] memory, bool) {
        return splitProofs(_data);
    }
}

contract SubmissionProxyMockTest is Test {
    SubmissionProxyMock submissionProxy;

    function setUp() public {
        submissionProxy = new SubmissionProxyMock();
    }

    function test_QuorumWithOneOracle() public {
        submissionProxy.addOracle(makeAddr("zero"));

        uint256 quorum_ = submissionProxy.callQuorum(1);
        assertEq(quorum_, 1);

        quorum_ = submissionProxy.callQuorum(100);
        assertEq(quorum_, 1);
    }

    function test_QuorumWithTwoOracles() public {
        submissionProxy.addOracle(makeAddr("zero"));
        submissionProxy.addOracle(makeAddr("one"));

        uint256 quorum_ = submissionProxy.callQuorum(1);
        assertEq(quorum_, 1);

        quorum_ = submissionProxy.callQuorum(50);
        assertEq(quorum_, 1);

        // STEP
        quorum_ = submissionProxy.callQuorum(51);
        assertEq(quorum_, 2);

        quorum_ = submissionProxy.callQuorum(100);
        assertEq(quorum_, 2);
    }

    function test_QuorumWithThreeOracles() public {
        submissionProxy.addOracle(makeAddr("zero"));
        submissionProxy.addOracle(makeAddr("one"));
        submissionProxy.addOracle(makeAddr("two"));

        uint256 quorum_ = submissionProxy.callQuorum(1);
        assertEq(quorum_, 1);

        quorum_ = submissionProxy.callQuorum(33);
        assertEq(quorum_, 1);

        // STEP
        quorum_ = submissionProxy.callQuorum(34);
        assertEq(quorum_, 2);

        quorum_ = submissionProxy.callQuorum(66);
        assertEq(quorum_, 2);

        // STEP
        quorum_ = submissionProxy.callQuorum(67);
        assertEq(quorum_, 3);

        quorum_ = submissionProxy.callQuorum(100);
        assertEq(quorum_, 3);
    }

    function test_SplitSignatureInvalidLength() public {
        bytes memory sig_ = new bytes(64);
        vm.expectRevert(SubmissionProxy.InvalidSignatureLength.selector);
        // FAIL - signature cannot be below or above 65 bytes long
        submissionProxy.callSplitSignature(sig_);

        sig_ = new bytes(66);
        vm.expectRevert(SubmissionProxy.InvalidSignatureLength.selector);
        // FAIL - signature cannot be below or above 65 bytes long
        submissionProxy.callSplitSignature(sig_);
    }

    function test_SplitSignature(uint8 _expectedV, bytes32 _expectedR, bytes32 _expectedS) public {
        bytes memory sig_ = abi.encodePacked(_expectedR, _expectedS, _expectedV);
        (uint8 v_, bytes32 r_, bytes32 s_) = submissionProxy.callSplitSignature(sig_);
        assertEq(v_, _expectedV);
        assertEq(r_, _expectedR);
        assertEq(s_, _expectedS);
    }

    function test_SplitProofsInvalidLength() public {
        bytes memory sig_ = new bytes(0);
        {
            // FAIL - signature length cannot be 0
            (, bool success) = submissionProxy.callSplitProofs(sig_);
            assertEq(success, false);
        }

        sig_ = new bytes(64);
        {
            // FAIL - signature length must be multiplier of 65
            (, bool success) = submissionProxy.callSplitProofs(sig_);
            assertEq(success, false);
        }

        sig_ = new bytes(66);
        {
            // FAIL - signature length must be multiplier of 65
            (, bool success) = submissionProxy.callSplitProofs(sig_);
            assertEq(success, false);
        }

        sig_ = new bytes(65);
        {
            // OK
            (, bool success) = submissionProxy.callSplitProofs(sig_);
            assertEq(success, true);
        }
    }

    function test_SplitProofsInvalidLength(
        uint8[2] memory _expectedV,
        bytes32[2] memory _expectedR,
        bytes32[2] memory _expectedS
    ) public {
        bytes memory sigOne_ = abi.encodePacked(_expectedR[0], _expectedS[0], _expectedV[0]);
        bytes memory sigTwo_ = abi.encodePacked(_expectedR[1], _expectedS[1], _expectedV[1]);

        bytes memory sig_ = abi.encodePacked(sigOne_, sigTwo_);

        (bytes[] memory proofs, bool success) = submissionProxy.callSplitProofs(sig_);
        assertEq(success, true);
        assertEq(sigOne_, proofs[0]);
        assertEq(sigTwo_, proofs[1]);
    }
}
