// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IAccount.sol";
import "./interfaces/ITypeAndVersion.sol";

/// @title Orakl Network Account
/// @author Bisonai
/// @notice Account contract represents a [regular] account type that
/// @notice is connected to a Prepayment (Prepayment.sol) contract.
/// @dev Account contract is deployed with a `Prepayment.addContract`
/// @dev call. The functions that modify the account state are allowed
/// @dev to to be called only through Prepayment contract, other ones
/// @dev are can be called on Account contract itself.
contract Account is IAccount, ITypeAndVersion {
    uint16 public constant MAX_CONSUMERS = 100;

    address private sPaymentSolution;
    uint64 private sAccId;

    // Account information
    address private sOwner; // Owner can fund/withdraw/cancel the acc
    address private sRequestedOwner; // For safely transferring acc ownership
    uint256 private sBalance; // Common $KLAY balance used for all consumer requests
    uint64 private sReqCount; // For fee tiers
    AccountType private sAccountType;
    uint256 private sStartTime; // subscription activation time
    uint256 private sPeriod; // subscription period
    uint256 private sPeriodReqCount; // number of allowed requests within a subscription period
    uint256 private sServiceFeeRatio; // discounted service fee denominated in basis points (e.g. 9500 -> 5 % discount)
    uint256 sSubscriptionPrice; // only for KLAY_SUBSCRIPTION

    address[] private sConsumers;

    /* consumer */
    /* nonce */
    mapping(address => uint64) private sConsumerToNonce;
    // period => request count
    mapping(uint256 => uint256) sSubReqCountHistory;
    mapping(uint256 => bool) sSubscriptionPaid; //  to track whether the payment has been already paid

    error TooManyConsumers();
    error MustBeRequestedOwner(address requestedOwner);
    error MustBeAccountOwner(address owner);
    error MustBePaymentSolution(address paymentSolution);
    error InsufficientBalance();
    error InvalidConsumer(address consumer);
    error BurnFeeFailed();
    error OperatorFeeFailed();
    error ProtocolFeeFailed();

    modifier onlyPaymentSolution() {
        if (msg.sender != sPaymentSolution) {
            revert MustBePaymentSolution(sPaymentSolution);
        }
        _;
    }

    constructor(uint64 accId, address owner, AccountType accType) {
        sAccId = accId;
        sOwner = owner;
        sPaymentSolution = msg.sender;
        sAccountType = accType;
    }

    receive() external payable {
        sBalance += msg.value;
    }

    /**
     * @inheritdoc IAccount
     */
    function getAccount()
        external
        view
        returns (
            uint256 balance,
            uint64 reqCount,
            address owner,
            address[] memory consumers,
            AccountType accType
        )
    {
        return (sBalance, sReqCount, sOwner, sConsumers, sAccountType);
    }

    /**
     * @inheritdoc IAccount
     */
    function getAccountId() external view returns (uint64) {
        return sAccId;
    }

    /**
     * @inheritdoc IAccount
     */
    function getBalance() external view returns (uint256) {
        return sBalance;
    }

    /**
     * @inheritdoc IAccount
     */
    function getReqCount() external view returns (uint64) {
        return sReqCount;
    }

    /**
     * @inheritdoc IAccount
     */
    function getOwner() external view returns (address) {
        return sOwner;
    }

    /**
     * @inheritdoc IAccount
     */
    function getConsumers() external view returns (address[] memory) {
        return sConsumers;
    }

    /**
     * @inheritdoc IAccount
     */
    function getRequestedOwner() external view returns (address) {
        return sRequestedOwner;
    }

    /**
     * @inheritdoc IAccount
     */
    function getNonce(address consumer) external view returns (uint64) {
        return sConsumerToNonce[consumer];
    }

    /**
     * @inheritdoc IAccount
     */
    function getPaymentSolution() external view returns (address) {
        return sPaymentSolution;
    }

    /**
     * @inheritdoc IAccount
     */
    function getAccountDetail() external view returns (uint256, uint256, uint256, uint256) {
        return (sStartTime, sPeriod, sPeriodReqCount, sSubscriptionPrice);
    }

    /**
     * @inheritdoc IAccount
     */
    function getSubscriptionPaid() external view returns (bool) {
        return sSubscriptionPaid[(block.timestamp - sStartTime) / sPeriod];
    }

    /**
     * @inheritdoc IAccount
     */
    function updateAccountDetail(
        uint256 startTime,
        uint256 period,
        uint256 reqPeriodCount,
        uint256 subscriptionPrice
    ) external onlyPaymentSolution {
        sStartTime = startTime;
        sPeriod = period;
        sPeriodReqCount = reqPeriodCount;
        sSubscriptionPrice = subscriptionPrice;
    }

    /**
     * @inheritdoc IAccount
     */
    function setSubscriptionPaid() external onlyPaymentSolution {
        sSubscriptionPaid[(block.timestamp - sStartTime) / sPeriod] = true;
    }

    /**
     * @inheritdoc IAccount
     */
    function getFeeRatio() external view returns (uint256) {
        return sServiceFeeRatio;
    }

    /**
     * @inheritdoc IAccount
     */
    function setFeeRatio(uint256 feeRatio) external onlyPaymentSolution {
        sServiceFeeRatio = feeRatio;
    }

    /**
     * @inheritdoc IAccount
     */
    function isValidReq() external view returns (bool) {
        if (
            sAccountType == AccountType.FIAT_SUBSCRIPTION ||
            sAccountType == AccountType.KLAY_SUBSCRIPTION
        ) {
            if (sSubReqCountHistory[(block.timestamp - sStartTime) / sPeriod] <= sPeriodReqCount) {
                return true;
            } else {
                return false;
            }
        }
        return true;
    }

    /**
     * @inheritdoc IAccount
     */
    function increaseNonce(address consumer) external onlyPaymentSolution returns (uint64) {
        uint64 nonce = sConsumerToNonce[consumer] + 1;
        sConsumerToNonce[consumer] = nonce;
        return nonce;
    }

    /**
     * @inheritdoc IAccount
     */
    function requestAccountOwnerTransfer(address newOwner) external onlyPaymentSolution {
        // Proposing the address(0) would never be claimable so no
        // need to check.
        if (sRequestedOwner != newOwner) {
            sRequestedOwner = newOwner;
        }
    }

    /**
     * @inheritdoc IAccount
     */
    function acceptAccountOwnerTransfer(address newOwner) external onlyPaymentSolution {
        if (sRequestedOwner != newOwner) {
            revert MustBeRequestedOwner(sRequestedOwner);
        }
        sOwner = newOwner;
        sRequestedOwner = address(0);
    }

    /**
     * @inheritdoc IAccount
     */
    function addConsumer(address consumer) external onlyPaymentSolution {
        // Already maxed, cannot add any more consumers.
        if (sConsumers.length >= MAX_CONSUMERS) {
            revert TooManyConsumers();
        }
        if (sConsumerToNonce[consumer] > 0) {
            // Idempotence - do nothing if already added. Ensures
            // uniqueness in sConsumers.
            return;
        }

        // Initialize the nonce to 1, indicating the consumer is
        // allocated.
        sConsumerToNonce[consumer] = 1;
        sConsumers.push(consumer);
    }

    /**
     * @inheritdoc IAccount
     */
    function removeConsumer(address consumer) external onlyPaymentSolution {
        if (sConsumerToNonce[consumer] == 0) {
            revert InvalidConsumer(consumer);
        }

        // Note bounded by MAX_CONSUMERS
        address[] memory consumers = sConsumers;
        uint256 consumersLength = consumers.length;
        uint256 lastConsumerIndex = consumersLength - 1;

        for (uint256 i = 0; i < consumersLength; ++i) {
            if (consumers[i] == consumer) {
                address last = consumers[lastConsumerIndex];
                // Storage write to preserve last element
                sConsumers[i] = last;
                // Storage remove last element
                sConsumers.pop();
                break;
            }
        }

        delete sConsumerToNonce[consumer];
    }

    /**
     * @inheritdoc IAccount
     */
    function withdraw(uint256 amount) external onlyPaymentSolution returns (bool, uint256) {
        uint256 balance = sBalance;

        if (balance < amount) {
            revert InsufficientBalance();
        }

        balance -= amount;
        sBalance = balance;

        (bool sent, ) = payable(sOwner).call{value: amount}("");

        return (sent, balance);
    }

    /**
     * @inheritdoc IAccount
     */
    function chargeFee(
        uint256 burnFee,
        uint256 protocolFee,
        address protocolFeeRecipient
    ) external onlyPaymentSolution {
        sReqCount += 1;
        sBalance -= (burnFee + protocolFee);

        if (burnFee > 0) {
            (bool sent, ) = address(0x000000000000000000000000000000000000dEaD).call{
                value: burnFee
            }("");
            if (!sent) {
                revert BurnFeeFailed();
            }
        }

        if (protocolFee > 0) {
            (bool sent, ) = protocolFeeRecipient.call{value: protocolFee}("");
            if (!sent) {
                revert ProtocolFeeFailed();
            }
        }
    }

    function chargeOperatorFee(
        uint256 operatorFee,
        address operatorFeeRecipient
    ) external onlyPaymentSolution {
        sBalance -= operatorFee;

        (bool sent, ) = operatorFeeRecipient.call{value: operatorFee}("");
        if (!sent) {
            revert OperatorFeeFailed();
        }
    }

    /**
     * @inheritdoc IAccount
     */
    function increaseSubReqCount() external onlyPaymentSolution {
        sSubReqCountHistory[(block.timestamp - sStartTime) / sPeriod] += 1;
    }

    /**
     * @inheritdoc IAccount
     */
    function cancelAccount(address to) external onlyPaymentSolution {
        selfdestruct(payable(to));
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Account v0.1";
    }
}
