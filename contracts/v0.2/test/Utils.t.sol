// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console2, console, Vm} from "forge-std/Test.sol";
import {UtilsScript} from "../deploy/Utils.s.sol";

contract UtilsTest is Test {
    UtilsScript config;

    function setUp() public {
        config = new UtilsScript();
    }

    function test_getAggregators() public {
        string memory dirPath = "/migration/local/Aggregator";
        string[] memory files = config.loadMigration(dirPath);
        if (files.length > 0) config.updateMigration(dirPath, files[0]);
    }
}
