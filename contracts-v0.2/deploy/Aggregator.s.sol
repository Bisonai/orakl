// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, console} from "forge-std/Script.sol";
import {Aggregator} from "../src/Aggregator.sol";
import {Utils} from "./Utils.sol";

contract AggregatorS is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        Utils config = new Utils();
        string memory dirPath = string.concat("/migration/", config.chainName(), "/Aggregator");
        string[] memory migrationFiles = config.loadMigration(dirPath);
        console.log("deploying...", migrationFiles.length, "contract");

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            vm.startBroadcast(deployerPrivateKey);
            Aggregator aggregator;
            string memory migrationFilePath = migrationFiles[i];
            bytes memory deployData = config.readJson(migrationFilePath, ".deploy");
            Utils.Deploy memory deployConfig = abi.decode(deployData, (Utils.Deploy));
            if (bytes(deployConfig.name).length > 0) {
                uint32 timeout = uint32(deployConfig.timeout);
                address validator = deployConfig.validator;
                uint8 decimals = uint8(deployConfig.decimals);
                string memory description = deployConfig.description;
                aggregator = new Aggregator(timeout, validator, decimals, description);
            }

            bytes memory changeOracleData = config.readJson(migrationFiles[i], ".changeOracles");
            Utils.ChangeOracles memory changeOracleConfig = abi.decode(
                changeOracleData,
                (Utils.ChangeOracles)
            );
            if (changeOracleConfig.minSubmissionCount > 0) {
                aggregator.changeOracles(
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
