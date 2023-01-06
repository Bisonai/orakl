// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/BlockhashStoreInterface.sol";
import "./interfaces/VRFCoordinatorInterface.sol";
import "./interfaces/TypeAndVersionInterface.sol";
import "./interfaces/PrepaymentInterface.sol";
import "./libraries/VRF.sol";
import "./ConfirmedOwner.sol";
import "./VRFConsumerBase.sol";
import "./interfaces/CoordinatorBaseInterface.sol";

contract VRFCoordinator is
    ConfirmedOwner,
    TypeAndVersionInterface,
    VRFCoordinatorInterface,
    CoordinatorBaseInterface
{
    error InvalidConsumer(uint64 accId, address consumer);
    error InvalidAccount();
    error MustBeAccOwner(address owner);
    error PendingRequestExists();
    error BalanceInvariantViolated(uint256 internalBalance, uint256 externalBalance); // Should never happen
    event FundsRecovered(address to, uint256 amount);

    // Set this maximum to 200 to give us a 56 block window to fulfill
    // the request before requiring the block hash feeder.
    uint16 public constant MAX_REQUEST_CONFIRMATIONS = 200;
    uint32 public constant MAX_NUM_WORDS = 500;
    // 5k is plenty for an EXTCODESIZE call (2600) + warm CALL (100)
    // and some arithmetic operations.
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;

    error InvalidRequestConfirmations(uint16 have, uint16 min, uint16 max);
    error GasLimitTooBig(uint32 have, uint32 want);
    error NumWordsTooBig(uint32 have, uint32 want);
    error ProvingKeyAlreadyRegistered(bytes32 keyHash);
    error NoSuchProvingKey(bytes32 keyHash);
    error InsufficientGasForConsumer(uint256 have, uint256 want);
    error NoCorrespondingRequest();
    error IncorrectCommitment();
    error BlockhashNotInStore(uint256 blockNum);
    error PaymentTooLarge();
    error Reentrant();
    struct RequestCommitment {
        uint64 blockNum;
        uint64 accId;
        uint32 callbackGasLimit;
        uint32 numWords;
        address sender;
    }

    /* keyHash */ /* oracle */
    mapping(bytes32 => address) private s_provingKeys;

    bytes32[] private s_provingKeyHashes;

    /* oracle */ /* KLAY balance */
    mapping(address => uint96) private s_withdrawableTokens;

    /* requestID */ /* commitment */
    mapping(uint256 => bytes32) private s_requestCommitments;

    event ProvingKeyRegistered(bytes32 keyHash, address indexed oracle);
    event ProvingKeyDeregistered(bytes32 keyHash, address indexed oracle);
    event RandomWordsRequested(
        bytes32 indexed keyHash,
        uint256 requestId,
        uint256 preSeed,
        uint64 indexed accId,
        uint16 minimumRequestConfirmations,
        uint32 callbackGasLimit,
        uint32 numWords,
        address indexed sender
    );
    event RandomWordsFulfilled(
        uint256 indexed requestId,
        uint256 outputSeed,
        uint96 payment,
        bool success
    );

    struct Config {
        uint16 minimumRequestConfirmations;
        uint32 maxGasLimit;
        // Reentrancy protection.
        bool reentrancyLock;
        // Gas to cover oracle payment after we calculate the payment.
        // We make it configurable in case those operations are repriced.
        uint32 gasAfterPaymentCalculation;
        bytes32 keyHash;
    }
    Config private s_config;
    FeeConfig private s_feeConfig;
    struct FeeConfig {
        // Flat fee charged per fulfillment in millionths of link
        // So fee range is [0, 2^32/10^6].
        uint32 fulfillmentFlatFeeLinkPPMTier1;
        uint32 fulfillmentFlatFeeLinkPPMTier2;
        uint32 fulfillmentFlatFeeLinkPPMTier3;
        uint32 fulfillmentFlatFeeLinkPPMTier4;
        uint32 fulfillmentFlatFeeLinkPPMTier5;
        uint24 reqsForTier2;
        uint24 reqsForTier3;
        uint24 reqsForTier4;
        uint24 reqsForTier5;
    }
    event ConfigSet(
        uint16 minimumRequestConfirmations,
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        FeeConfig feeConfig
    );

    PrepaymentInterface Prepayment;

    constructor(
        //  address blockhashStore
        PrepaymentInterface prepayment
    ) ConfirmedOwner(msg.sender) {
        // BLOCKHASH_STORE = BlockhashStoreInterface(blockhashStore);
        // set prepayment
        Prepayment = prepayment;
    }

    /**
     * @notice Registers a proving key to an oracle.
     * @param oracle address of the oracle
     * @param publicProvingKey key that oracle can use to submit vrf fulfillments
     */
    function registerProvingKey(
        address oracle,
        uint256[2] calldata publicProvingKey
    ) external onlyOwner {
        bytes32 kh = hashOfKey(publicProvingKey);
        if (s_provingKeys[kh] != address(0)) {
            revert ProvingKeyAlreadyRegistered(kh);
        }
        s_provingKeys[kh] = oracle;
        s_provingKeyHashes.push(kh);
        emit ProvingKeyRegistered(kh, oracle);
    }

    /**
     * @notice Deregisters a proving key to an oracle.
     * @param publicProvingKey key that oracle can use to submit vrf fulfillments
     */
    function deregisterProvingKey(uint256[2] calldata publicProvingKey) external onlyOwner {
        bytes32 kh = hashOfKey(publicProvingKey);
        address oracle = s_provingKeys[kh];
        if (oracle == address(0)) {
            revert NoSuchProvingKey(kh);
        }
        delete s_provingKeys[kh];
        for (uint256 i = 0; i < s_provingKeyHashes.length; i++) {
            if (s_provingKeyHashes[i] == kh) {
                bytes32 last = s_provingKeyHashes[s_provingKeyHashes.length - 1];
                // Copy last element and overwrite kh to be deleted with it
                s_provingKeyHashes[i] = last;
                s_provingKeyHashes.pop();
            }
        }
        emit ProvingKeyDeregistered(kh, oracle);
    }

    /**
     * @notice Returns the proving key hash key associated with this public key
     * @param publicKey the key to return the hash of
     */
    function hashOfKey(uint256[2] memory publicKey) public pure returns (bytes32) {
        return keccak256(abi.encode(publicKey));
    }

    /**
     * @notice Sets the configuration of the vrf coordinator
     * @param minimumRequestConfirmations global min for request confirmations
     * @param maxGasLimit global max for request gas limit
     * @param gasAfterPaymentCalculation gas used in doing accounting after completing the gas measurement
     * @param feeConfig fee tier configuration
     */
    function setConfig(
        uint16 minimumRequestConfirmations,
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        bytes32 keyHash,
        FeeConfig memory feeConfig
    ) external onlyOwner {
        if (minimumRequestConfirmations > MAX_REQUEST_CONFIRMATIONS) {
            revert InvalidRequestConfirmations(
                minimumRequestConfirmations,
                minimumRequestConfirmations,
                MAX_REQUEST_CONFIRMATIONS
            );
        }
        s_config = Config({
            minimumRequestConfirmations: minimumRequestConfirmations,
            maxGasLimit: maxGasLimit,
            gasAfterPaymentCalculation: gasAfterPaymentCalculation,
            reentrancyLock: false,
            keyHash: keyHash
        });
        s_feeConfig = feeConfig;
        emit ConfigSet(
            minimumRequestConfirmations,
            maxGasLimit,
            gasAfterPaymentCalculation,
            s_feeConfig
        );
    }

    function getConfig()
        external
        view
        returns (
            uint16 minimumRequestConfirmations,
            uint32 maxGasLimit,
            uint32 gasAfterPaymentCalculation
        )
    {
        return (
            s_config.minimumRequestConfirmations,
            s_config.maxGasLimit,
            s_config.gasAfterPaymentCalculation
        );
    }

    function getFeeConfig()
        external
        view
        returns (
            uint32 fulfillmentFlatFeeLinkPPMTier1,
            uint32 fulfillmentFlatFeeLinkPPMTier2,
            uint32 fulfillmentFlatFeeLinkPPMTier3,
            uint32 fulfillmentFlatFeeLinkPPMTier4,
            uint32 fulfillmentFlatFeeLinkPPMTier5,
            uint24 reqsForTier2,
            uint24 reqsForTier3,
            uint24 reqsForTier4,
            uint24 reqsForTier5
        )
    {
        return (
            s_feeConfig.fulfillmentFlatFeeLinkPPMTier1,
            s_feeConfig.fulfillmentFlatFeeLinkPPMTier2,
            s_feeConfig.fulfillmentFlatFeeLinkPPMTier3,
            s_feeConfig.fulfillmentFlatFeeLinkPPMTier4,
            s_feeConfig.fulfillmentFlatFeeLinkPPMTier5,
            s_feeConfig.reqsForTier2,
            s_feeConfig.reqsForTier3,
            s_feeConfig.reqsForTier4,
            s_feeConfig.reqsForTier5
        );
    }

    /**
     * @notice Recover link sent with transfer instead of transferAndCall.
     * @param to address to send link to
     */
    /**
     * @inheritdoc VRFCoordinatorInterface
     */
    function getRequestConfig() external view override returns (uint16, uint32, bytes32[] memory) {
        return (s_config.minimumRequestConfirmations, s_config.maxGasLimit, s_provingKeyHashes);
    }

    /**
     * @inheritdoc VRFCoordinatorInterface
     */
    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint16 requestConfirmations,
        uint32 callbackGasLimit,
        uint32 numWords
    ) public override nonReentrant returns (uint256) {
        // Input validation using the account storage.
        // call to prepayment contract
        address owner = Prepayment.getAccountOwner(accId);
        if (owner == address(0)) {
            revert InvalidAccount();
        }

        // Its important to ensure that the consumer is in fact who they say they
        // are, otherwise they could use someone else's account balance.
        // A nonce of 0 indicates consumer is not allocated to the sub.
        uint64 currentNonce = Prepayment.getNonce(msg.sender, accId);
        if (currentNonce == 0) {
            revert InvalidConsumer(accId, msg.sender);
        }

        // Input validation using the config storage word.
        if (
            requestConfirmations < s_config.minimumRequestConfirmations ||
            requestConfirmations > MAX_REQUEST_CONFIRMATIONS
        ) {
            revert InvalidRequestConfirmations(
                requestConfirmations,
                s_config.minimumRequestConfirmations,
                MAX_REQUEST_CONFIRMATIONS
            );
        }

        // No lower bound on the requested gas limit. A user could request 0
        // and they would simply be billed for the proof verification and wouldn't be
        // able to do anything with the random value.
        if (callbackGasLimit > s_config.maxGasLimit) {
            revert GasLimitTooBig(callbackGasLimit, s_config.maxGasLimit);
        }

        if (numWords > MAX_NUM_WORDS) {
            revert NumWordsTooBig(numWords, MAX_NUM_WORDS);
        }

        // Note we do not check whether the keyHash is valid to save gas.
        // The consequence for users is that they can send requests
        // for invalid keyHashes which will simply not be fulfilled.
        //uint64 nonce = currentNonce + 1;
        (uint256 requestId, uint256 preSeed) = computeRequestId(
            keyHash,
            msg.sender,
            accId,
            currentNonce + 1
        );

        s_requestCommitments[requestId] = keccak256(
            abi.encode(requestId, block.number, accId, callbackGasLimit, numWords, msg.sender)
        );
        emit RandomWordsRequested(
            keyHash,
            requestId,
            preSeed,
            accId,
            requestConfirmations,
            callbackGasLimit,
            numWords,
            msg.sender
        );

        //increase nonce for consumer
        Prepayment.increaseNonce(msg.sender, accId);

        return requestId;
    }

    /**
     * @notice Get request commitment
     * @param requestId id of request
     * @dev used to determine if a request is fulfilled or not
     */
    function getCommitment(uint256 requestId) external view returns (bytes32) {
        return s_requestCommitments[requestId];
    }

    function computeRequestId(
        bytes32 keyHash,
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256, uint256) {
        uint256 preSeed = uint256(keccak256(abi.encode(keyHash, sender, accId, nonce)));
        return (uint256(keccak256(abi.encode(keyHash, preSeed))), preSeed);
    }

    /**
     * @dev calls target address with exactly gasAmount gas and data as calldata
     * or reverts if at least gasAmount gas is not available.
     */
    function callWithExactGas(
        uint256 gasAmount,
        address target,
        bytes memory data
    ) private returns (bool success) {
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

    function getRandomnessFromProof(
        VRF.Proof memory proof,
        RequestCommitment memory rc
    ) private view returns (bytes32 keyHash, uint256 requestId, uint256 randomness) {
        keyHash = hashOfKey(proof.pk);
        // Only registered proving keys are permitted.
        address oracle = s_provingKeys[keyHash];
        if (oracle == address(0)) {
            revert NoSuchProvingKey(keyHash);
        }
        requestId = uint256(keccak256(abi.encode(keyHash, proof.seed)));
        bytes32 commitment = s_requestCommitments[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }
        if (
            commitment !=
            keccak256(
                abi.encode(
                    requestId,
                    rc.blockNum,
                    rc.accId,
                    rc.callbackGasLimit,
                    rc.numWords,
                    rc.sender
                )
            )
        ) {
            revert IncorrectCommitment();
        }

        bytes32 blockHash = blockhash(rc.blockNum);
        // FIXME
        //if (blockHash == bytes32(0)) {
        //  blockHash = BLOCKHASH_STORE.getBlockhash(rc.blockNum);
        //  if (blockHash == bytes32(0)) {
        //    revert BlockhashNotInStore(rc.blockNum);
        //  }
        //}

        // The seed actually used by the VRF machinery, mixing in the blockhash
        bytes memory actualSeed = abi.encodePacked(
            keccak256(abi.encodePacked(proof.seed, blockHash))
        );
        randomness = VRF.randomValueFromVRFProof(proof, actualSeed); // Reverts on failure
    }

    /*
     * @notice Compute fee based on the request count
     * @param reqCount number of requests
     * @return feePPM fee in LINK PPM
     */
    function getFeeTier(uint64 reqCount) public view returns (uint32) {
        FeeConfig memory fc = s_feeConfig;
        if (0 <= reqCount && reqCount <= fc.reqsForTier2) {
            return fc.fulfillmentFlatFeeLinkPPMTier1;
        }
        if (fc.reqsForTier2 < reqCount && reqCount <= fc.reqsForTier3) {
            return fc.fulfillmentFlatFeeLinkPPMTier2;
        }
        if (fc.reqsForTier3 < reqCount && reqCount <= fc.reqsForTier4) {
            return fc.fulfillmentFlatFeeLinkPPMTier3;
        }
        if (fc.reqsForTier4 < reqCount && reqCount <= fc.reqsForTier5) {
            return fc.fulfillmentFlatFeeLinkPPMTier4;
        }
        return fc.fulfillmentFlatFeeLinkPPMTier5;
    }

    /*
     * @notice Fulfill a randomness request
     * @param proof contains the proof and randomness
     * @param rc request commitment pre-image, committed to at request time
     * @return payment amount billed to the account
     * @dev simulated offchain to determine if sufficient balance is present to fulfill the request
     */
    function fulfillRandomWords(
        VRF.Proof memory proof,
        RequestCommitment memory rc
    ) external nonReentrant returns (uint96) {
        /* uint256 startGas = gasleft(); */
        (, /* bytes32 keyHash */ uint256 requestId, uint256 randomness) = getRandomnessFromProof(
            proof,
            rc
        );

        uint256[] memory randomWords = new uint256[](rc.numWords);
        for (uint256 i = 0; i < rc.numWords; i++) {
            randomWords[i] = uint256(keccak256(abi.encode(randomness, i)));
        }

        delete s_requestCommitments[requestId];
        VRFConsumerBase v;
        bytes memory resp = abi.encodeWithSelector(
            v.rawFulfillRandomWords.selector,
            requestId,
            randomWords
        );
        // Call with explicitly the amount of callback gas requested
        // Important to not let them exhaust the gas budget and avoid oracle payment.
        // Do not allow any non-view/non-pure coordinator functions to be called
        // during the consumers callback code via reentrancyLock.
        // Note that callWithExactGas will revert if we do not have sufficient gas
        // to give the callee their requested amount.
        s_config.reentrancyLock = true;
        bool success = callWithExactGas(rc.callbackGasLimit, rc.sender, resp);
        s_config.reentrancyLock = false;
        // We want to charge users exactly for how much gas they use in their callback.
        // The gasAfterPaymentCalculation is meant to cover these additional operations where we
        // decrement the account balance and increment the oracles withdrawable balance.
        // We also add the flat link fee to the payment amount.
        // Its specified in millionths of link, if s_config.fulfillmentFlatFeeLinkPPM = 1
        // 1 link / 1e6 = 1e18 juels / 1e6 = 1e12 juels.
        // FIXME fix payment
        //uint96 payment = 0;

        /* (uint96 balance, uint64 reqCount, address owner, address[] memory consumers) = Prepayment */
        /*     .getAccount(rc.accId); */

        // uint96 payment = calculatePaymentAmount(
        //   startGas,
        //   s_config.gasAfterPaymentCalculation,
        //   getFeeTier(reqCount),
        //   tx.gasprice
        // );
        // if (balance < payment) {
        //   revert InsufficientBalance();
        // }
        uint96 payment = 10 ** 17;
        Prepayment.chargeFee(rc.accId, payment);
        //s_withdrawableTokens[s_provingKeys[rc.keyHash]] += payment; FIXME  Does need to do this?
        // Include payment in the event for tracking costs.
        emit RandomWordsFulfilled(requestId, randomness, payment, success);
        return payment;
    }

    //// Get the amount of gas used for fulfillment
    // function calculatePaymentAmount(
    //   uint256 startGas,
    //   uint256 gasAfterPaymentCalculation,
    //   uint32 fulfillmentFlatFeeLinkPPM,
    //   uint256 weiPerUnitGas
    // ) internal view returns (uint96) {
    //   // (1e18 juels/link) (wei/gas * gas) = juels
    //   uint256 paymentNoFee = (1e18 * weiPerUnitGas * (gasAfterPaymentCalculation + startGas - gasleft()));
    //   uint256 fee = 1e12 * uint256(fulfillmentFlatFeeLinkPPM);
    //   if (paymentNoFee > (1e27 - fee)) {
    //     revert PaymentTooLarge(); // Payment + fee cannot be more than all of the link in existence.
    //   }
    //   return uint96(paymentNoFee + fee);
    // }

    function pendingRequestExists(
        uint64 accId,
        address consumer,
        uint64 nonce
    ) public view override returns (bool) {
        for (uint256 j = 0; j < s_provingKeyHashes.length; j++) {
            (uint256 reqId, ) = computeRequestId(s_provingKeyHashes[j], consumer, accId, nonce);
            if (s_requestCommitments[reqId] != 0) {
                return true;
            }
        }
        return false;
    }

    receive() external payable {}

    /**
     * @inheritdoc VRFCoordinatorInterface
     */
    function requestRandomWordsPayment(
        uint16 requestConfirmations,
        uint32 callbackGasLimit,
        uint32 numWords
    ) external payable override returns (uint256) {
        require(msg.value > 0, "Insufficient balance");
        // create account
        uint64 accId = Prepayment.createAccount();
        Prepayment.addConsumer(accId, msg.sender);
        uint256 requestId = requestRandomWords(
            s_config.keyHash,
            accId,
            requestConfirmations,
            callbackGasLimit,
            numWords
        );
        Prepayment.deposit{value: msg.value}(accId);
        return requestId;
    }

    modifier nonReentrant() {
        if (s_config.reentrancyLock) {
            revert Reentrant();
        }
        _;
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "VRFCoordinator 0.1";
    }
}
