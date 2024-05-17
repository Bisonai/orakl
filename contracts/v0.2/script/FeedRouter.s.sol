// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, stdJson, console} from "forge-std/Script.sol";
import {UtilsScript} from "./Utils.s.sol";
import {FeedRouter} from "../src/FeedRouter.sol";
import {strings} from "solidity-stringutils/strings.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

contract DeployFeedRouter is Script {
    using stdJson for string;
    using strings for *;

    UtilsScript config;

    function setUp() public {
        config = new UtilsScript();
    }

    function run() public {
        string memory dirPath = string.concat("/migration/", config.chainName(), "/FeedRouter");
        string[] memory migrationFiles = config.loadMigration(dirPath);

        vm.startBroadcast(vm.envUint("PRIVATE_KEY"));

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            string memory migrationFilePath = migrationFiles[i];
            string memory json = vm.readFile(migrationFilePath);
            console.log("Migration File", migrationFilePath);

            bool result = executeMigration(json);
            if (!result) {
                console.log("Migration failed");
                continue;
            }
            config.updateMigration(dirPath, migrationFilePath);
        }
        vm.stopBroadcast();
    }

    function updateProxies(FeedRouter feedRouter, bytes memory rawJson) internal {
        UtilsScript.UpdateProxyBulkConstructor[] memory updateProxyBulkConstructors =
            abi.decode(rawJson, (UtilsScript.UpdateProxyBulkConstructor[]));
        string[] memory _feedNames = new string[](updateProxyBulkConstructors.length);
        address[] memory _proxyAddresses = new address[](updateProxyBulkConstructors.length);

        for (uint256 j = 0; j < updateProxyBulkConstructors.length; j++) {
            UtilsScript.UpdateProxyBulkConstructor memory updateProxyBulkConstructor = updateProxyBulkConstructors[j];

            _feedNames[j] = updateProxyBulkConstructor.feedName;
            _proxyAddresses[j] = updateProxyBulkConstructor.proxyAddress;
            console.log(
                "(Proxy Prepared)", updateProxyBulkConstructor.feedName, updateProxyBulkConstructor.proxyAddress
            );
        }
        feedRouter.updateProxyBulk(_feedNames, _proxyAddresses);
        console.log("(Proxies Updated)");
    }

    function executeMigration(string memory json) public returns (bool) {
        FeedRouter feedRouter;
        bool useExisting = vm.keyExists(json, ".address");
        bool deploy = vm.keyExists(json, ".deploy");

        // if both .deploy and .address exists, use deployed contract address
        if (deploy) {
            console.log("Deploying FeedRouter");
            feedRouter = new FeedRouter();
            config.storeAddress("FeedRouter", address(feedRouter));
            console.log("(FeedRouter Deployed)", address(feedRouter));
        } else if (useExisting) {
            bytes memory feedRouterAddressRaw = json.parseRaw(".address");
            address feedRouterAddress = abi.decode(feedRouterAddressRaw, (address));
            feedRouter = FeedRouter(feedRouterAddress);
            console.log("(Use existing FeedRouter)", address(feedRouter));
        } else {
            console.log("FeedRouter not found, skipping deploy");
            return false;
        }

        bool updateProxyBulk = vm.keyExists(json, ".updateProxyBulk");
        if (updateProxyBulk) {
            bytes memory feedRouterProxyConstructorsRaw = json.parseRaw(".updateProxyBulk");
            updateProxies(feedRouter, feedRouterProxyConstructorsRaw);
        }
        return true;
    }
}
