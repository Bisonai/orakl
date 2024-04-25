// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, stdJson, VmSafe, console} from "forge-std/Script.sol";
import {strings} from "solidity-stringutils/strings.sol";

contract UtilsScript is Script {
    using stdJson for string;
    using strings for *;

    string MIGRATION_LOCK_FILE_NAME = "migration.lock";

    struct FeedConstructor {
        uint256 decimals;
        string description;
    }

    struct ChangeOracles {
        address[] addedAdmins;
        address[] added;
        uint256 maxSubmissionCount;
        uint256 minSubmissionCount;
        address[] removed;
        uint256 restartDelay;
    }

    function chainName() public view returns (string memory chain) {
        if (block.chainid == 1001) {
            return "baobab";
        } else if (block.chainid == 8217) {
            return "cypress";
        }
        return "local";
    }

    function loadMigration(string memory dirPath) public returns (string[] memory fileNames) {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, dirPath);
        VmSafe.DirEntry[] memory files = vm.readDir(path);

        string memory lockFilePath = string.concat(path, "/", MIGRATION_LOCK_FILE_NAME);
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

    function readJson(string memory filePath, string memory key) public view returns (bytes memory) {
        string memory json = vm.readFile(filePath);
        return json.parseRaw(key);
    }
}
