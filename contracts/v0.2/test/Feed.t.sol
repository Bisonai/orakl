// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Feed} from "../src/Feed.sol";

contract FeedTest is Test {
    Feed public feed;
    uint8 decimals = 18;
    string description = "Test Feed";
    uint256 timestamp = 1706170779;
    address[] removed;
    address[] added;

    function setUp() public {
        vm.warp(timestamp);
        feed = new Feed(decimals, description);
    }

    function test_AddAndRemoveOracle() public {
	address alice = makeAddr('alice');
	address bob = makeAddr('bob');

        added.push(alice);
        added.push(bob);

	feed.changeOracles(removed, added);
        assertEq(feed.getOracles().length, 2);

	// remove what has been added (switched parameters)
	feed.changeOracles(added, removed);
        assertEq(feed.getOracles().length, 0);
    }

    function test_RemoveNonexistantOracle() public {
	address alice = makeAddr('alice');

	removed.push(alice);
	vm.expectRevert(Feed.OracleNotEnabled.selector);
	feed.changeOracles(removed, added);
    }

    function test_AddOracleTwice() public {
	address alice = makeAddr('alice');

	added.push(alice);
	feed.changeOracles(removed, added);
	vm.expectRevert(Feed.OracleAlreadyEnabled.selector);
	feed.changeOracles(removed, added);
    }

    /* function test_SubmitAndReadResponse() public { */
    /*     clear(); */
    /*     oracleAdd.push(address(0)); */

    /*     changeOracles(); */
    /*     vm.prank(address(0)); */
    /*     vm.expectEmit(true, true, false, true); */
    /*     emit NewRound(1, address(0), timestamp); */
    /*     feed.submit(10); */
    /*     (, int256 answer,,,) = feed.latestRoundData(); */
    /*     assertEq(answer, 10); */
    /* } */

    /* function test_RevertWith_TooManyOracles() public { */
    /*     uint256 maxOracle = feed.MAX_ORACLE_COUNT(); */
    /*     for (uint32 i = 1; i < maxOracle; i++) { */
    /*         address add = address(bytes20(keccak256(abi.encodePacked(i)))); */
    /*         oracleAdd.push(add); */
    /*         feed.changeOracles(oracleRemove, oracleAdd, i, i, 0); */
    /*         oracleAdd.pop(); */
    /*     } */
    /*     vm.expectRevert(Feed.TooManyOracles.selector); */
    /*     oracleAdd.push(address(0)); */
    /*     feed.changeOracles(oracleRemove, oracleAdd, uint32(maxOracle), uint32(maxOracle), 0); */
    /* } */

    /* function test_RevertWith_MinSubmissionGtMaxSubmission() public { */
    /*     vm.expectRevert(Feed.MinSubmissionGtMaxSubmission.selector); */
    /*     feed.changeOracles(oracleRemove, oracleAdd, 1, 0, 0); */
    /* } */

    /* function test_RevertWith_MaxSubmissionGtOracleNum() public { */
    /*     vm.expectRevert(Feed.MaxSubmissionGtOracleNum.selector); */
    /*     feed.changeOracles(oracleRemove, oracleAdd, 0, 1, 0); */
    /* } */

    /* function test_RevertWith_RestartDelayExceedOracleNum() public { */
    /*     uint32 minSubmissionCount = 0; */
    /*     uint32 maxSubmissionCount = 1; */
    /*     uint32 restartDelay = 1; */
    /*     vm.expectRevert(Feed.RestartDelayExceedOracleNum.selector); */
    /*     oracleAdd.push(address(0)); */
    /*     feed.changeOracles(oracleRemove, oracleAdd, minSubmissionCount, maxSubmissionCount, restartDelay); */
    /* } */

    /* function test_RevertWith_MinSubmissionZero() public { */
    /*     uint32 minSubmissionCount = 0; */
    /*     uint32 maxSubmissionCount = 1; */
    /*     uint32 restartDelay = 0; */
    /*     vm.expectRevert(Feed.MinSubmissionZero.selector); */
    /*     oracleAdd.push(address(0)); */
    /*     feed.changeOracles(oracleRemove, oracleAdd, minSubmissionCount, maxSubmissionCount, restartDelay); */
    /* } */

    /* function test_RevertWith_PrevRoundNotSupersedable() public { */
    /*     bool authorized = true; */
    /*     uint32 deplay = 0; */
    /*     feed.setRequesterPermissions(address(0), authorized, deplay); */
    /*     oracleAdd.push(address(1)); */
    /*     oracleAdd.push(address(2)); */
    /*     oracleAdd.push(address(3)); */
    /*     feed.changeOracles(oracleRemove, oracleAdd, 2, 3, 0); */
    /*     vm.prank(address(1)); */
    /*     feed.submit(321); */
    /*     vm.expectRevert(Feed.PrevRoundNotSupersedable.selector); */
    /*     vm.prank(address(0)); */
    /*     feed.requestNewRound(); */
    /* } */

    /* function test_currentRoundStartedAt() public { */
    /*     oracleAdd.push(address(0)); */
    /*     changeOracles(); */
    /*     for (uint256 i = 1; i <= 2; i++) { */
    /*         vm.warp(timestamp + i); */
    /*         vm.prank(address(0)); */
    /*         feed.submit(321); */
    /*         uint256 startedAt = feed.currentRoundStartedAt(); */
    /*         assertEq(startedAt, timestamp + i); */
    /*         (uint80 roundId,,,,) = feed.latestRoundData(); */
    /*         assertEq(roundId, i); */
    /*     } */
    /* } */

    /* function test_validateOracleRound() public { */
    /*     vm.expectRevert("not enabled oracle"); */
    /*     feed.submit(321); */
    /*     oracleAdd.push(address(0)); */
    /*     changeOracles(); */
    /* } */
}
