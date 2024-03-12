// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {Feed} from "../src/Feed.sol";
import {UtilsScript} from "./Utils.s.sol";

contract FeedScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        UtilsScript config = new UtilsScript();
        string memory dirPath = string.concat("/migration/", config.chainName(), "/Feed");
        string[] memory migrationFiles = config.loadMigration(dirPath);
        console.log("deploying...", migrationFiles.length, "contract");

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            vm.startBroadcast(deployerPrivateKey);
            Feed feed;
            string memory migrationFilePath = migrationFiles[i];
            bytes memory deployData = config.readJson(migrationFilePath, ".deploy");
            UtilsScript.Deploy memory deployConfig = abi.decode(deployData, (UtilsScript.Deploy));
            if (bytes(deployConfig.name).length > 0) {
                uint32 timeout = uint32(deployConfig.timeout);
                uint8 decimals = uint8(deployConfig.decimals);
                string memory description = deployConfig.description;
                feed = new Feed(timeout, decimals, description);
            }

            bytes memory changeOracleData = config.readJson(migrationFiles[i], ".changeOracles");
            UtilsScript.ChangeOracles memory changeOracleConfig =
                abi.decode(changeOracleData, (UtilsScript.ChangeOracles));
            if (changeOracleConfig.minSubmissionCount > 0) {
                feed.changeOracles(
                    changeOracleConfig.removed,
                    changeOracleConfig.added,
                    uint32(changeOracleConfig.minSubmissionCount),
                    uint32(changeOracleConfig.maxSubmissionCount),
                    uint32(changeOracleConfig.restartDelay)
                );
            }
            vm.stopBroadcast();
            config.updateMigration(dirPath, migrationFilePath);
        }
        console.log("deployed", migrationFiles.length, "contract");
    }
}
