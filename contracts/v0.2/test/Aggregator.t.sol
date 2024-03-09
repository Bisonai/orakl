// SPDX-License-Identifier: UNLINCENSED
pragma solidity ^0.8.24;

import {Test, console2, console} from "forge-std/Test.sol";
import {Aggregator} from "../src/Aggregator.sol";

contract AggregatorTest is Test {
    Aggregator public aggregator;
    uint32 timeout = 10;
    address validator = address(0);
    uint8 decimals = 18;
    string description = "Test Aggregator";

    address[] oracleRemove;
    address[] oracleAdd;

    uint256 timestamp = 1706170779;

    event AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt);

    event NewRound(uint256 indexed roundId, address indexed startedBy, uint256 startedAt);

    function clear() internal {
        for (uint256 i = 0; i < oracleRemove.length; i++) {
            oracleRemove.pop();
        }
        for (uint256 i = 0; i < oracleAdd.length; i++) {
            oracleAdd.pop();
        }
    }

    function changeOracles() internal {
        uint256 maxSubmission = aggregator.getOracles().length + oracleAdd.length - oracleRemove.length;
        uint32 minSubmission = 1;
        if (maxSubmission > 2) minSubmission = 2;
        aggregator.changeOracles(oracleRemove, oracleAdd, minSubmission, uint32(maxSubmission), 0);
    }

    function setUp() public {
        vm.warp(timestamp);
        aggregator = new Aggregator(timeout, decimals, description);
    }

    function test_AddAndRemoveOracle() public {
        clear();
        oracleAdd.push(address(0));
        oracleAdd.push(address(1));
        changeOracles();
        assertEq(aggregator.getOracles().length, 2);

        oracleAdd.pop();
        oracleAdd.pop();

        oracleRemove.push(address(1));
        changeOracles();
        assertEq(aggregator.getOracles().length, 1);
    }

    function testFail_RemoveOracleNoExisted() public {
        clear();
        oracleAdd.push(address(0));
        changeOracles();

        assertEq(aggregator.getOracles().length, 1);
        oracleRemove.push(address(2));
        changeOracles();
    }

    function testFail_AddOracleTwice() public {
        clear();
        oracleAdd.push(address(0));
        changeOracles();

        assertEq(aggregator.getOracles().length, 1);
        oracleAdd.push(address(0));
        changeOracles();
    }

    function test_SubmitAndReadResponse() public {
        clear();
        oracleAdd.push(address(0));

        changeOracles();
        vm.prank(address(0));
        vm.expectEmit(true, true, false, true);
        emit NewRound(1, address(0), timestamp);
        aggregator.submit(10);
        (, int256 answer,,,) = aggregator.latestRoundData();
        assertEq(answer, 10);
    }

    function test_RevertWith_TooManyOracles() public {
        uint256 maxOracle = aggregator.MAX_ORACLE_COUNT();
        for (uint32 i = 1; i < maxOracle; i++) {
            address add = address(bytes20(keccak256(abi.encodePacked(i))));
            oracleAdd.push(add);
            aggregator.changeOracles(oracleRemove, oracleAdd, i, i, 0);
            oracleAdd.pop();
        }
        vm.expectRevert(Aggregator.TooManyOracles.selector);
        oracleAdd.push(address(0));
        aggregator.changeOracles(oracleRemove, oracleAdd, uint32(maxOracle), uint32(maxOracle), 0);
    }

    function test_RevertWith_MinSubmissionGtMaxSubmission() public {
        vm.expectRevert(Aggregator.MinSubmissionGtMaxSubmission.selector);
        aggregator.changeOracles(oracleRemove, oracleAdd, 1, 0, 0);
    }

    function test_RevertWith_MaxSubmissionGtOracleNum() public {
        vm.expectRevert(Aggregator.MaxSubmissionGtOracleNum.selector);
        aggregator.changeOracles(oracleRemove, oracleAdd, 0, 1, 0);
    }

    function test_RevertWith_RestartDelayExceedOracleNum() public {
        uint32 minSubmissionCount = 0;
        uint32 maxSubmissionCount = 1;
        uint32 restartDelay = 1;
        vm.expectRevert(Aggregator.RestartDelayExceedOracleNum.selector);
        oracleAdd.push(address(0));
        aggregator.changeOracles(oracleRemove, oracleAdd, minSubmissionCount, maxSubmissionCount, restartDelay);
    }

    function test_RevertWith_MinSubmissionZero() public {
        uint32 minSubmissionCount = 0;
        uint32 maxSubmissionCount = 1;
        uint32 restartDelay = 0;
        vm.expectRevert(Aggregator.MinSubmissionZero.selector);
        oracleAdd.push(address(0));
        aggregator.changeOracles(oracleRemove, oracleAdd, minSubmissionCount, maxSubmissionCount, restartDelay);
    }

    function test_RevertWith_PrevRoundNotSupersedable() public {
        bool authorized = true;
        uint32 deplay = 0;
        aggregator.setRequesterPermissions(address(0), authorized, deplay);
        oracleAdd.push(address(1));
        oracleAdd.push(address(2));
        oracleAdd.push(address(3));
        aggregator.changeOracles(oracleRemove, oracleAdd, 2, 3, 0);
        vm.prank(address(1));
        aggregator.submit(321);
        vm.expectRevert(Aggregator.PrevRoundNotSupersedable.selector);
        vm.prank(address(0));
        aggregator.requestNewRound();
    }

    function test_currentRoundStartedAt() public {
        oracleAdd.push(address(0));
        changeOracles();
        for (uint256 i = 1; i <= 2; i++) {
            vm.warp(timestamp + i);
            vm.prank(address(0));
            aggregator.submit(321);
            uint256 startedAt = aggregator.currentRoundStartedAt();
            assertEq(startedAt, timestamp + i);
            (uint80 roundId,,,,) = aggregator.latestRoundData();
            assertEq(roundId, i);
        }
    }

    function test_validateOracleRound() public {
        vm.expectRevert("not enabled oracle");
        aggregator.submit(321);
        oracleAdd.push(address(0));
        changeOracles();
    }
}
