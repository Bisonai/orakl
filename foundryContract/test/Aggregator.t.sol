// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2} from "forge-std/Test.sol";
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

    event AnswerUpdated(
        int256 indexed current,
        uint256 indexed roundId,
        uint256 updatedAt
    );

    event NewRound(
        uint256 indexed roundId,
        address indexed startedBy,
        uint256 startedAt
    );

    function clear() internal {
        for (uint i = 0; i < oracleRemove.length; i++) {
            oracleRemove.pop();
        }
        for (uint i = 0; i < oracleAdd.length; i++) {
            oracleAdd.pop();
        }
    }

    function changeOracles() internal {
        uint256 maxSubmission = aggregator.getOracles().length +
            oracleAdd.length -
            oracleRemove.length;
        uint32 minSubmission = 1;
        if (maxSubmission > 2) minSubmission = 2;

        aggregator.changeOracles(
            oracleRemove,
            oracleAdd,
            minSubmission,
            uint32(maxSubmission),
            0
        );
    }

    function setUp() public {
        vm.warp(timestamp);
        aggregator = new Aggregator(timeout, validator, decimals, description);
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
        oracleAdd.push(address(1));
        oracleAdd.push(address(2));

        changeOracles();

        vm.prank(address(0));
        vm.expectEmit(true, true, false, true);
        emit NewRound(1, address(0), timestamp);
        aggregator.submit(1, 10);

        vm.prank(address(1));
        vm.expectEmit(true, true, false, true);
        emit AnswerUpdated(10, 1, timestamp);
        aggregator.submit(1, 11);

        vm.prank(address(2));
        vm.expectEmit(true, true, false, true);
        emit AnswerUpdated(11, 1, timestamp);
        aggregator.submit(1, 12);

        (, int256 answer, , , ) = aggregator.latestRoundData();
        assertEq(answer, 11);
    }
}
