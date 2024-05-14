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

    function setUp() public {}
    function run() public {
        FeedRouter feedRouter;
        UtilsScript config = new UtilsScript();
        string memory dirPath = string.concat("/migration/", config.chainName(), "/FeedRouter");
        string[] memory migrationFiles = config.loadMigration(dirPath);

        vm.startBroadcast(vm.envUint("PRIVATE_KEY"));

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            string memory migrationFilePath = migrationFiles[i];
            string memory json = vm.readFile(migrationFilePath);
            console.log("Migration File", migrationFilePath);

            bool useExisting = vm.keyExists(json, ".address");
            bool deploy = vm.keyExists(json, ".deploy");

            // if both .deploy and .address exists, use deployed contract address
            if (deploy) {
                console.log("Deploying FeedRouter");
                feedRouter = new FeedRouter();
                console.log("(FeedRouter Deployed)", address(feedRouter));
            }else if (useExisting) {
                bytes memory feedRouterAddressRaw = json.parseRaw(".address");
                address feedRouterAddress = abi.decode(feedRouterAddressRaw, (address));
                feedRouter = FeedRouter(feedRouterAddress);
                console.log("(Use existing FeedRouter)", address(feedRouter));
            }else {
                console.log("FeedRouter not found, skipping deploy");
                continue;
            }

            bool updateProxyBulk = vm.keyExists(json, ".updateProxyBulk");
            if (updateProxyBulk) {
                bytes memory feedRouterProxyConstructorsRaw = json.parseRaw(".updateProxyBulk.proxies");
                updateProxies(feedRouter, feedRouterProxyConstructorsRaw);
            }

            config.updateMigration(dirPath, migrationFilePath);
        }
        vm.stopBroadcast();
    }

    function updateProxies(FeedRouter feedRouter, bytes memory rawJson) internal {
        UtilsScript.UpdateProxyBulkProxyConstructor[] memory updateProxyBulkProxyConstructors = abi.decode(rawJson, (UtilsScript.UpdateProxyBulkProxyConstructor[]));
        string[] memory _feedNames = new string[](updateProxyBulkProxyConstructors.length);
        address[] memory _proxyAddresses = new address[](updateProxyBulkProxyConstructors.length);

        for (uint256 j = 0; j < updateProxyBulkProxyConstructors.length; j++) {
            UtilsScript.UpdateProxyBulkProxyConstructor memory updateProxyBulkProxyConstructor = updateProxyBulkProxyConstructors[j];

            _feedNames[j] = updateProxyBulkProxyConstructor.feedName;
            _proxyAddresses[j] = updateProxyBulkProxyConstructor.proxyAddress;
            console.log("(Proxy Prepared)", updateProxyBulkProxyConstructor.feedName, updateProxyBulkProxyConstructor.proxyAddress);
        }
        feedRouter.updateProxyBulk(_feedNames, _proxyAddresses);
        console.log("(Proxies Updated)");
    }
}