// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/IAccount.sol";

abstract contract CoordinatorBase is Ownable, ICoordinatorBase {
    // 5k is plenty for an EXTCODESIZE call (2600) + warm CALL (100)
    // and some arithmetic operations.
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;

    address[] public sOracles;

    /* requestID */
    /* commitment */
    mapping(uint256 => bytes32) internal sRequestIdToCommitment;

    /* requestID */
    /* owner */
    mapping(uint256 => address) internal sRequestOwner;

    IPrepayment internal sPrepayment;

    struct Config {
        uint32 maxGasLimit;
        bool reentrancyLock;
        // Gas to cover oracle payment after we calculate the payment.
        // We make it configurable in case those operations are repriced.
        uint32 gasAfterPaymentCalculation;
    }
    Config internal sConfig;

    FeeConfig private sFeeConfig;

    error Reentrant();
    error NoCorrespondingRequest();
    error NotRequestOwner();
    error OracleAlreadyRegistered(address oracle);
    error NoSuchOracle(address oracle);
    error RefundFailure();
    error InvalidConsumer(uint64 accId, address consumer);
    error IncorrectCommitment();
    error GasLimitTooBig(uint32 have, uint32 want);
    error InsufficientPayment(uint256 have, uint256 want);

    event ConfigSet(uint32 maxGasLimit, uint32 gasAfterPaymentCalculation, FeeConfig feeConfig);
    event RequestCanceled(uint256 indexed requestId);

    modifier nonReentrant() {
        if (sConfig.reentrancyLock) {
            revert Reentrant();
        }
        _;
    }

    /**
     * @inheritdoc ICoordinatorBase
     */
    function setConfig(
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        FeeConfig memory feeConfig
    ) external onlyOwner {
        sConfig = Config({
            maxGasLimit: maxGasLimit,
            gasAfterPaymentCalculation: gasAfterPaymentCalculation,
            reentrancyLock: false
        });
        sFeeConfig = feeConfig;
        emit ConfigSet(maxGasLimit, gasAfterPaymentCalculation, sFeeConfig);
    }

    function getConfig()
        external
        view
        returns (uint32 maxGasLimit, uint32 gasAfterPaymentCalculation)
    {
        return (sConfig.maxGasLimit, sConfig.gasAfterPaymentCalculation);
    }

    function getFeeConfig()
        external
        view
        returns (
            uint32 fulfillmentFlatFeeKlayPPMTier1,
            uint32 fulfillmentFlatFeeKlayPPMTier2,
            uint32 fulfillmentFlatFeeKlayPPMTier3,
            uint32 fulfillmentFlatFeeKlayPPMTier4,
            uint32 fulfillmentFlatFeeKlayPPMTier5,
            uint24 reqsForTier2,
            uint24 reqsForTier3,
            uint24 reqsForTier4,
            uint24 reqsForTier5
        )
    {
        return (
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier1,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier2,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier3,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier4,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier5,
            sFeeConfig.reqsForTier2,
            sFeeConfig.reqsForTier3,
            sFeeConfig.reqsForTier4,
            sFeeConfig.reqsForTier5
        );
    }

    function getPrepaymentAddress() external view returns (address) {
        return address(sPrepayment);
    }

    /**
     * @inheritdoc ICoordinatorBase
     */
    function getCommitment(uint256 requestId) external view returns (bytes32) {
        return sRequestIdToCommitment[requestId];
    }

    /**
     * @inheritdoc ICoordinatorBase
     */
    function cancelRequest(uint256 requestId) external {
        if (!isValidRequestId(requestId)) {
            revert NoCorrespondingRequest();
        }

        if (sRequestOwner[requestId] != msg.sender) {
            revert NotRequestOwner();
        }

        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];

        emit RequestCanceled(requestId);
    }

    function estimateFee(
        uint64 reqCount,
        uint8 numSubmission,
        uint32 callbackGasLimit
    ) public view returns (uint256) {
        uint256 serviceFee = calculateServiceFee(reqCount) * numSubmission;
        uint256 maxGasCost = tx.gasprice * callbackGasLimit;
        return serviceFee + maxGasCost;
    }

    function estimateFeeByAcc(
        uint64 reqCount,
        uint8 numSubmission,
        uint32 callbackGasLimit,
        uint64 accId,
        IAccount.AccountType accType
    ) public view returns (uint256) {
        (, , , uint256 subscriptionPrice) = sPrepayment.getAccountDetail(accId);
        uint256 minBalance;
        if (accType == IAccount.AccountType.KLAY_DISCOUNT) {
            uint256 feeRatio = sPrepayment.getFeeRatio(accId);
            uint256 baseFee = estimateFee(reqCount, numSubmission, callbackGasLimit);
            minBalance = (baseFee * feeRatio) / 10_000;
        } else if (accType == IAccount.AccountType.KLAY_SUBSCRIPTION) {
            if (!sPrepayment.getSubscriptionPaid(accId)) {
                minBalance = subscriptionPrice;
            }
        } else if (accType == IAccount.AccountType.KLAY_REGULAR) {
            minBalance = estimateFee(reqCount, numSubmission, callbackGasLimit);
        }
        return minBalance;
    }

    /**
     * @notice Calculate service fee based on tier system of the
     * coordinator.
     */
    function calculateServiceFee(uint64 reqCount) internal view returns (uint256) {
        uint32 fulfillmentFlatFeeKlayPPM = getFeeTier(reqCount);
        return 1e12 * uint256(fulfillmentFlatFeeKlayPPM);
    }

    function calculateGasCost(uint256 startGas) internal view returns (uint256) {
        return tx.gasprice * (sConfig.gasAfterPaymentCalculation + startGas - gasleft());
    }

    /**
     * @notice Compute fee based on the request count
     * @param reqCount number of requests
     * @return feePPM fee in KLAY PPM
     */
    function getFeeTier(uint64 reqCount) internal view returns (uint32) {
        FeeConfig memory fc = sFeeConfig;
        if (0 <= reqCount && reqCount <= fc.reqsForTier2) {
            return fc.fulfillmentFlatFeeKlayPPMTier1;
        }
        if (fc.reqsForTier2 < reqCount && reqCount <= fc.reqsForTier3) {
            return fc.fulfillmentFlatFeeKlayPPMTier2;
        }
        if (fc.reqsForTier3 < reqCount && reqCount <= fc.reqsForTier4) {
            return fc.fulfillmentFlatFeeKlayPPMTier3;
        }
        if (fc.reqsForTier4 < reqCount && reqCount <= fc.reqsForTier5) {
            return fc.fulfillmentFlatFeeKlayPPMTier4;
        }
        return fc.fulfillmentFlatFeeKlayPPMTier5;
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

    function isValidRequestId(uint256 requestId) internal view returns (bool) {
        if (sRequestIdToCommitment[requestId] != 0) {
            return true;
        } else {
            return false;
        }
    }

    function serviceFeeByAcc(uint64 accId, uint32 numSubmission) internal returns (uint256) {
        (, uint64 reqCount, , , IAccount.AccountType accType) = sPrepayment.getAccount(accId);
        (, , , uint256 subscriptionPrice) = sPrepayment.getAccountDetail(accId);

        if (accType == IAccount.AccountType.FIAT_SUBSCRIPTION) {
            sPrepayment.increaseSubReqCount(accId);
            return 0;
        } else {
            if (accType == IAccount.AccountType.KLAY_SUBSCRIPTION) {
                sPrepayment.increaseSubReqCount(accId);
            }
            uint256 serviceFee = calculateServiceFee(reqCount) * numSubmission;
            if (accType == IAccount.AccountType.KLAY_SUBSCRIPTION) {
                if (!sPrepayment.getSubscriptionPaid(accId)) {
                    serviceFee = subscriptionPrice;
                    sPrepayment.setSubscriptionPaid(accId);
                } else {
                    return 0;
                }
            } else if (accType == IAccount.AccountType.KLAY_DISCOUNT) {
                serviceFee = (serviceFee * sPrepayment.getFeeRatio(accId)) / 10_000;
            }
            return serviceFee;
        }
    }
}
