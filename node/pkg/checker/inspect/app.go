package inspect

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/alert"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/secrets"

	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

const (
	CallbackGasLimit = uint32(500_000)
	DefaultCheckCron = "0 0 * * *"

	RequestVRFSignature = `requestVRF(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) public returns (uint256)`
	RequestRRSignature = `requestRR(
        uint64 accId,
        uint32 callbackGasLimit
    ) public returns (uint256)`

	ReadRandomWordSignature = "sRandomWord() public view returns (uint256)"
	ReadResponseSignature   = "sResponse() public view returns (uint128)"
	VRFRequestIdSignature   = "vrfRequestId() public view returns (uint256)"
	RRRequestIdSignature    = "rrRequestId() public view returns (uint256)"
)

type Inspector struct {
	chainHelper *helper.ChainHelper
	address     string
	accountID   string
	keyHash     string
}

func Start(ctx context.Context) error {
	inspector, err := Setup(ctx)
	if err != nil {
		return err
	}

	c := cron.New()
	_, err = c.AddFunc("0 0 * * *", func() {
		result, inspectErr := inspector.Inspect(ctx)
		if inspectErr != nil {
			log.Error().Err(inspectErr).Msg("Error inspecting")
			alert.SlackAlert("inspector failed due to error: " + inspectErr.Error())
			return
		}

		log.Info().Str("Result", result).Msg("Inspect result")
		alert.SlackAlert(result)
	})
	if err != nil {
		return err
	}
	<-ctx.Done()

	return nil
}

func Setup(ctx context.Context) (*Inspector, error) {
	pk := secrets.GetSecret("INSPECTOR_PK")
	if pk == "" {
		return nil, errors.New("missing INSPECTOR_PK")
	}

	accountID := os.Getenv("ACC_ID")
	if accountID == "" {
		return nil, errors.New("missing ACC_ID")
	}

	keyHash := os.Getenv("VRF_KEYHASH")
	if keyHash == "" {
		return nil, errors.New("missing VRF_KEYHASH")
	}

	consumerContractAddress := os.Getenv("INSPECT_CONSUMER_ADDRESS")
	if consumerContractAddress == "" {
		return nil, errors.New("missing INSPECT_CONSUMER_ADDRESS")
	}

	chainHelper, err := helper.NewChainHelper(ctx, helper.WithReporterPk(pk), helper.WithoutAdditionalProviderUrls())
	if err != nil {
		return nil, err
	}

	inspector := NewInspector(chainHelper, consumerContractAddress, accountID, keyHash)
	return inspector, nil
}

func NewInspector(chainHelper *helper.ChainHelper, address string, accountID string, keyHash string) *Inspector {
	return &Inspector{
		chainHelper: chainHelper,
		address:     address,
		accountID:   accountID,
		keyHash:     keyHash,
	}
}

func (i *Inspector) Inspect(ctx context.Context) (string, error) {
	msg := "[Inspector]\n"
	inspectVRFResult, err := i.inspectVRF(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed VRF inspection")
		return "", err
	}
	msg += inspectVRFResult

	inspectRRResult, err := i.inspectRR(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed RR inspection")
		return "", err
	}

	msg += inspectRRResult
	return msg, nil
}

func (i *Inspector) inspectVRF(ctx context.Context) (string, error) {
	randomWordBefore, err := i.readBigIntValueFromContract(ctx, ReadRandomWordSignature)
	if err != nil {
		return "", err
	}
	requestIDBefore, err := i.readBigIntValueFromContract(ctx, VRFRequestIdSignature)
	if err != nil {
		return "", err
	}
	log.Debug().Any("randomWordBefore", randomWordBefore).Any("VRFrequestIDBefore", requestIDBefore).Msg("before")

	err = i.requestVRF(ctx)
	if err != nil {
		return "", err
	}
	log.Debug().Msg("request success")

	// follows same timeout as js inspector
	time.Sleep(5 * time.Second)

	randomWordAfter, err := i.readBigIntValueFromContract(ctx, ReadRandomWordSignature)
	if err != nil {
		return "", err
	}
	requestIDAfter, err := i.readBigIntValueFromContract(ctx, VRFRequestIdSignature)
	if err != nil {
		return "", err
	}
	log.Debug().Any("randomWordBefore", randomWordBefore).Any("VRFrequestIDBefore", requestIDBefore).Msg("after")

	if randomWordBefore.Cmp(randomWordAfter) == 0 || requestIDBefore.Cmp(requestIDAfter) == 0 {
		return "Inspector VRF: Failed\n", nil
	}

	return "Inspector VRF: Success\n", nil
}

func (i *Inspector) inspectRR(ctx context.Context) (string, error) {
	responseBefore, err := i.readBigIntValueFromContract(ctx, ReadResponseSignature)
	if err != nil {
		return "", err
	}
	requestIDBefore, err := i.readBigIntValueFromContract(ctx, RRRequestIdSignature)
	if err != nil {
		return "", err
	}
	log.Debug().Any("responseBefore", responseBefore).Any("RRrequestIDBefore", requestIDBefore).Msg("before")

	err = i.requestRR(ctx)
	if err != nil {
		return "", err
	}
	log.Debug().Msg("request success")

	// follows same timeout as js inspector
	time.Sleep(5 * time.Second)

	responseAfter, err := i.readBigIntValueFromContract(ctx, ReadResponseSignature)
	if err != nil {
		return "", err
	}
	requestIDAfter, err := i.readBigIntValueFromContract(ctx, RRRequestIdSignature)
	if err != nil {
		return "", err
	}
	log.Debug().Any("responseBefore", responseBefore).Any("RRrequestIDBefore", requestIDBefore).Msg("after")

	if responseBefore.Cmp(responseAfter) == 0 || requestIDBefore.Cmp(requestIDAfter) == 0 {
		return "Inspector RR: Failed\n", nil
	}

	return "Inspector RR: Success\n", nil
}

func (i *Inspector) requestVRF(ctx context.Context) error {
	keyHashBytes := klaytncommon.Hex2Bytes(strings.TrimPrefix(i.keyHash, "0x"))
	keyHash := [32]byte{}
	copy(keyHash[:], keyHashBytes)

	accountID, err := strconv.ParseUint(i.accountID, 10, 64)
	if err != nil {
		return err
	}

	numwords := uint32(1)

	tx, err := i.chainHelper.MakeDirectTx(ctx, i.address, RequestVRFSignature, keyHash, accountID, CallbackGasLimit, numwords)
	if err != nil {
		return err
	}

	return i.chainHelper.Submit(ctx, tx)
}

func (i *Inspector) requestRR(ctx context.Context) error {
	accountID, err := strconv.ParseUint(i.accountID, 10, 64)
	if err != nil {
		return err
	}

	tx, err := i.chainHelper.MakeDirectTx(ctx, i.address, RequestRRSignature, accountID, CallbackGasLimit)
	if err != nil {
		return err
	}

	return i.chainHelper.Submit(ctx, tx)
}

func (i *Inspector) readBigIntValueFromContract(ctx context.Context, signature string) (*big.Int, error) {
	rawResult, err := i.chainHelper.ReadContract(ctx, i.address, signature)
	if err != nil {
		return nil, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return nil, errors.New("failed to parse result")
	}

	return rawResultSlice[0].(*big.Int), nil
}
