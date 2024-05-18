// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, stdJson, VmSafe, console} from "forge-std/Script.sol";
import {strings} from "solidity-stringutils/strings.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

contract UtilsScript is Script {
    using Strings for uint256;
    using Strings for address;
    using stdJson for string;
    using strings for *;

    string MIGRATION_LOCK_FILE_NAME = "migration.lock";

    struct FeedConstructor {
        uint256 decimals;
        string description;
    }

    struct UpdateProxyBulkConstructor {
        string feedName;
        address proxyAddress;
    }

    struct SetProofThresholdConstructor {
        string name;
        uint8 threshold;
    }

    struct UpdateFeedConstructor {
        address feedAddress;
        string name;
    }

    struct FeedProxyUpdateConstructor {
        address feedAddress;
        address feedProxyAddress;
    }

    function chainName() public view returns (string memory chain) {
        if (block.chainid == 1001) {
            return "baobab";
        } else if (block.chainid == 8217) {
            return "cypress";
        }
        return "localhost";
    }

    function loadMigration(string memory dirPath) public returns (string[] memory fileNames) {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, dirPath);
        bool pathExists = vm.isDir(path);
        if (!pathExists) {
            vm.createDir(path, true);
        }

        VmSafe.DirEntry[] memory files = vm.readDir(path);
        string memory lockFilePath = string.concat(path, "/", MIGRATION_LOCK_FILE_NAME);

        bool lockFileExists = vm.isFile(lockFilePath);
        if (!lockFileExists) {
            console.log("lock file not exists, create one: ", lockFilePath);
            vm.writeFile(lockFilePath, "");
            console.log("lock file created");
        }

        string memory migrationFileName;
        string memory migrationFilePath;
        uint256 fileCount = 0;
        for (uint256 i = 0; i < files.length; i++) {
            VmSafe.DirEntry memory entry = files[i];
            if (keccak256(abi.encodePacked(lockFilePath)) == keccak256(abi.encodePacked(entry.path))) continue;
            bool fileExisted = false;
            migrationFileName = vm.readLine(lockFilePath);
            migrationFilePath = string.concat(path, "/", migrationFileName);

            while (bytes(migrationFileName).length > 0) {
                if (keccak256(abi.encodePacked(migrationFilePath)) == keccak256(abi.encodePacked(entry.path))) {
                    fileExisted = true;
                    break;
                }
                migrationFileName = vm.readLine(lockFilePath);
                migrationFilePath = string.concat(path, "/", migrationFileName);
            }
            if (fileExisted) continue;

            fileCount++;
        }
        string[] memory migrationFilePaths = new string[](fileCount);
        fileCount = 0;
        vm.closeFile(lockFilePath);
        for (uint256 i = 0; i < files.length; i++) {
            VmSafe.DirEntry memory entry = files[i];
            if (keccak256(abi.encodePacked(lockFilePath)) == keccak256(abi.encodePacked(entry.path))) continue;
            bool fileExisted = false;
            migrationFileName = vm.readLine(lockFilePath);
            migrationFilePath = string.concat(path, "/", migrationFileName);

            while (bytes(migrationFileName).length > 0) {
                if (keccak256(abi.encodePacked(migrationFilePath)) == keccak256(abi.encodePacked(entry.path))) {
                    fileExisted = true;
                    break;
                }
                migrationFileName = vm.readLine(lockFilePath);
                migrationFilePath = string.concat(path, "/", migrationFileName);
            }
            if (fileExisted) continue;
            migrationFilePaths[fileCount] = entry.path;
            fileCount++;
        }
        return migrationFilePaths;
    }

    function updateMigration(string memory dirPath, string memory fileName) public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, dirPath, "/", MIGRATION_LOCK_FILE_NAME);
        strings.slice memory s = strings.toSlice(fileName);
        strings.slice memory delim = strings.toSlice("/");
        string[] memory parts = new string[](s.count(delim) + 1);
        for (uint256 i = 0; i < parts.length; i++) {
            parts[i] = strings.split(s, delim).toString();
        }
        vm.writeLine(path, parts[parts.length - 1]);
    }

    function storeAddress(string memory contractName, address _address) public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, "/addresses/", chainName());
        if (!vm.isDir(path)) {
            vm.createDir(path, true);
        }

        string memory filePath = string.concat(path, "/", string.concat(contractName, ".json"));
        string memory json = string.concat("{ \"address\": \"", _address.toHexString(), "\" }");

        vm.writeJson(json, filePath);
    }

    function storeFeedAddress(string memory feedName, address feedAddress, address feedProxyAddress) public {
        string memory feedContractName = string.concat("Feed_", feedName);
        string memory feedProxyContractName = string.concat("FeedProxy_", feedName);

        storeAddress(feedContractName, feedAddress);
        storeAddress(feedProxyContractName, feedProxyAddress);
    }

    function readJson(string memory filePath, string memory key) public view returns (bytes memory) {
        string memory json = vm.readFile(filePath);
        return json.parseRaw(key);
    }
}
