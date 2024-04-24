// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";
import {IFeed} from "../src/interfaces/IFeed.sol";

contract SubmissionProxyMock is SubmissionProxy {
    function callQuorum(uint8 _threshold) external view returns (uint8) {
	return quorum(_threshold);
    }
}

contract SubmissionProxyMockTest is Test {
    SubmissionProxyMock submissionProxy;

    function setUp() public {
        submissionProxy = new SubmissionProxyMock();
    }

    function test_QuorumWithOneOracle() public {
	submissionProxy.addOracle(makeAddr("zero"));

	uint8 quorum_ = submissionProxy.callQuorum(1);
	assertEq(quorum_, 1);

	quorum_ = submissionProxy.callQuorum(100);
	assertEq(quorum_, 1);
    }

    function test_QuorumWithTwoOracles() public {
	submissionProxy.addOracle(makeAddr("zero"));
	submissionProxy.addOracle(makeAddr("one"));

	uint8 quorum_ = submissionProxy.callQuorum(1);
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

	uint8 quorum_ = submissionProxy.callQuorum(1);
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
}
