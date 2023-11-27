// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

abstract contract L2EndpointBase {
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;
    bool private sReentrancyLock;
    struct RequestInfo {
        address owner;
        uint32 callbackGasLimit;
    }

    mapping(address => bool) sSubmitters;
    uint256 public sSubmitterCount;
    mapping(uint256 => RequestInfo) internal sRequestDetail;

    error Reentrant();
    error InvalidSubmitter(address submitter);

    constructor() {}

    modifier nonReentrant() {
        if (sReentrancyLock) {
            revert Reentrant();
        }
        _;
    }

    function setReentrancy(bool value) internal {
        sReentrancyLock = value;
    }

    /**
     * @dev calls target address with exactly gasAmount gas and data as calldata
     * or reverts if at least gasAmount gas is not available.
     */
    function callWithExactGas(
        uint256 gasAmount,
        address target,
        bytes memory data
    ) internal returns (bool success) {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            let g := gas()
            // Compute g -= GAS_FOR_CALL_EXACT_CHECK and check for underflow
            // The gas actually passed to the callee is min(gasAmount, 63//64*gas available).
            // We want to ensure that we revert if gasAmount >  63//64*gas available
            // as we do not want to provide them with less, however that check itself costs
            // gas.  GAS_FOR_CALL_EXACT_CHECK ensures we have at least enough gas to be able
            // to revert if gasAmount >  63//64*gas available.
            if lt(g, GAS_FOR_CALL_EXACT_CHECK) {
                revert(0, 0)
            }
            g := sub(g, GAS_FOR_CALL_EXACT_CHECK)
            // if g - g//64 <= gasAmount, revert
            // (we subtract g//64 because of EIP-150)
            if iszero(gt(sub(g, div(g, 64)), gasAmount)) {
                revert(0, 0)
            }
            // solidity calls check that a contract actually exists at the destination, so we do the same
            if iszero(extcodesize(target)) {
                revert(0, 0)
            }
            // call and return whether we succeeded. ignore return data
            // call(gas,addr,value,argsOffset,argsLength,retOffset,retLength)
            success := call(gasAmount, target, 0, add(data, 0x20), mload(data), 0, 0)
        }
        return success;
    }
}
