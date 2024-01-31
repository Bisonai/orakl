// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2, console, Vm} from "forge-std/Test.sol";
import {Utils} from "../deploy/Utils.sol";

contract UtilsTest is Test {
    Utils config;

    function setUp() public {
        config = new Utils();
    }

    function test_getAggregators() public {
        string memory dirPath = "/migration/local/Aggregator";
        string[] memory files = config.loadMigration(dirPath);
        if (files.length > 0) config.updateMigration(dirPath, files[0]);
    }
}
