// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeed.sol";
import {ITypeAndVersion} from "./interfaces/ITypeAndVersion.sol";

contract Feed is Ownable, IFeed, ITypeAndVersion {
    uint8 public override decimals;
    string public override description;

    uint64 private latestRoundId;
    mapping(uint64 => Round) internal rounds;

    address[] private oracles;
    mapping(address => bool) private whitelist;

    struct Round {
        int256 answer;
        uint64 updatedAt;
    }

    event OraclePermissionsUpdated(address indexed oracle, bool indexed whitelisted);
    event FeedUpdated(int256 indexed answer, uint256 indexed roundId, uint256 updatedAt);

    error OnlyOracle();
    error OracleAlreadyEnabled();
    error OracleNotEnabled();
    error NoDataPresent();

    modifier onlyOracle() {
        if (!whitelist[msg.sender]) {
	    revert OnlyOracle();
	}
        _;
    }

    constructor(uint8 _decimals, string memory _description) Ownable(msg.sender) {
        decimals = _decimals;
        description = _description;
    }

    function submit(int256 _answer) external onlyOracle {
        uint64 roundId_ = latestRoundId + 1;

	rounds[roundId_].answer = _answer;
        rounds[roundId_].updatedAt = uint64(block.timestamp);

	emit FeedUpdated(_answer, roundId_, block.timestamp);
	latestRoundId = roundId_;
    }

    function changeOracles(
        address[] calldata _removed,
        address[] calldata _added
    ) external onlyOwner {
        for (uint256 i = 0; i < _removed.length; i++) {
            removeOracle(_removed[i]);
        }

        for (uint256 i = 0; i < _added.length; i++) {
	    if (_added[i] == address(0)) {
		continue;
	    }
            addOracle(_added[i]);
        }
    }

    function getOracles() external view returns (address[] memory) {
        return oracles;
    }

    function latestRoundData()
        external
        view
        virtual
        override
        returns (uint64 roundId, int256 answer, uint256 updatedAt)
    {
        return getRoundData(latestRoundId);
    }

    function latestRoundUpdatedAt() external view returns (uint256) {
        Round storage round = rounds[latestRoundId];
        return round.updatedAt;
    }

    function getRoundData(uint64 _roundId)
        public
        view
        virtual
        override
        returns (uint64 roundId, int256 answer, uint256 updatedAt)
    {
        Round memory r = rounds[_roundId];

        if (r.updatedAt == 0) {
            revert NoDataPresent();
        }

        return (_roundId, r.answer, r.updatedAt);
    }

    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Feed v0.2";
    }

    function addOracle(address _oracle) private {
        if (whitelist[_oracle]) {
            revert OracleAlreadyEnabled();
        }

	whitelist[_oracle] = true;
        oracles.push(_oracle);
        emit OraclePermissionsUpdated(_oracle, true);
    }

    function removeOracle(address _oracle) private {
        if (!whitelist[_oracle]) {
            revert OracleNotEnabled();
        }

	whitelist[_oracle] = false;
	for (uint i = 0; i < oracles.length; i++) {
	    if (oracles[i] == _oracle) {
		oracles[i] = oracles[oracles.length - 1];
		oracles.pop();
		break;
	    }
	}

        emit OraclePermissionsUpdated(_oracle, false);
    }
}
