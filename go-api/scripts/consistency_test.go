package scripts

import (
	"encoding/json"
	"go-api/adapter"
	"go-api/aggregate"
	"go-api/aggregator"
	"go-api/apierr"
	"go-api/chain"
	"go-api/data"
	"go-api/feed"
	"go-api/listener"
	"go-api/proxy"
	"go-api/reporter"
	"go-api/service"
	"go-api/utils"
	"go-api/vrf"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// assumes that both node server and go server is both up in local env
// http://127.0.0.1:3000/api/v1/
const (
	NODE_PORT = "3000"
	GO_PORT   = "3111"
	NODE_URL  = "http://127.0.0.1:" + NODE_PORT + "/api/v1"
	GO_URL    = "http://127.0.0.1:" + GO_PORT + "/api/v1"
)

type AdapterInsertModel struct {
	_AdapterInsertModel
	FEEDS []feed.FeedInsertModel `json:"feeds"`
}

type _AdapterInsertModel struct {
	ADAPTER_HASH string `db:"adapter_hash" json:"adapterHash" validate:"required"`
	NAME         string `db:"name" json:"name" validate:"required"`
	DECIMALS     int    `db:"decimals" json:"decimals" validate:"required"`
}

var insertedService service.ServiceModel
var insertedChain chain.ChainModel
var insertedAdapter adapter.AdapterModel
var insertedAggregator aggregator.AggregatorResultModel
var insertedAggregate aggregate.AggregateModel
var insertedFeeds []feed.FeedModel
var insertedError apierr.ErrorModel
var insertedListener listener.ListenerModel
var insertedProxy proxy.ProxyModel
var insertedReporter reporter.ReporterModel
var insertedVrf vrf.VrfModel
var insertedData data.DataResultModel

func TestConsistency(t *testing.T) {
	beforeAll()

	TestVrfConsistency(t)
	TestServiceConsistency(t)
	TestReporterConsistency(t)
	TestProxyConsistency(t)
	TestListenerConsistency(t)
	TestFeedConsistency(t)
	TestDataConsistency(t)
	TestChainConsistency(t)
	TestApierrConsistency(t)
	TestAggregatorConsistency(t)
	TestAggregateConsistency(t)
	TestAdapterConsistency(t)

	defer cleanup()
}

func TestVrfConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]vrf.VrfModel](NODE_URL+"/vrf", "GET", map[string]any{"chain": insertedChain.Name})
	readAllFromGoApi, _ := utils.UrlRequest[[]vrf.VrfModel](GO_URL+"/vrf", "GET", map[string]any{"chain": insertedChain.Name})
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[vrf.VrfModel](NODE_URL+"/vrf/"+insertedVrf.VrfKeyId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[vrf.VrfModel](GO_URL+"/vrf/"+insertedVrf.VrfKeyId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestServiceConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]service.ServiceModel](NODE_URL+"/service", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]service.ServiceModel](GO_URL+"/service", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[service.ServiceModel](NODE_URL+"/service/"+insertedService.ServiceId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[service.ServiceModel](GO_URL+"/service/"+insertedService.ServiceId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestReporterConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]reporter.ReporterModel](NODE_URL+"/reporter", "GET", map[string]any{"chain": insertedChain.Name, "service": insertedService.Name})
	readAllFromGoApi, _ := utils.UrlRequest[[]reporter.ReporterModel](GO_URL+"/reporter", "GET", map[string]any{"chain": insertedChain.Name, "service": insertedService.Name})
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[reporter.ReporterModel](NODE_URL+"/reporter/"+insertedReporter.ReporterId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[reporter.ReporterModel](GO_URL+"/reporter/"+insertedReporter.ReporterId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestProxyConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]proxy.ProxyModel](NODE_URL+"/proxy", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]proxy.ProxyModel](GO_URL+"/proxy", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[proxy.ProxyModel](NODE_URL+"/proxy/"+insertedProxy.Id.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[proxy.ProxyModel](GO_URL+"/proxy/"+insertedProxy.Id.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestListenerConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]listener.ListenerModel](NODE_URL+"/listener", "GET", map[string]any{"chain": insertedChain.Name, "service": insertedService.Name})
	readAllFromGoApi, _ := utils.UrlRequest[[]listener.ListenerModel](GO_URL+"/listener", "GET", map[string]any{"chain": insertedChain.Name, "service": insertedService.Name})
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[listener.ListenerModel](NODE_URL+"/listener/"+insertedListener.ListenerId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[listener.ListenerModel](GO_URL+"/listener/"+insertedListener.ListenerId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestFeedConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]feed.FeedModel](NODE_URL+"/feed", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]feed.FeedModel](GO_URL+"/feed", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[feed.FeedModel](NODE_URL+"/feed/"+insertedFeeds[0].FeedId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[feed.FeedModel](GO_URL+"/feed/"+insertedFeeds[0].FeedId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestDataConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]data.DataResultModel](NODE_URL+"/data", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]data.DataResultModel](GO_URL+"/data", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[data.DataResultModel](NODE_URL+"/data/"+insertedData.DataId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[data.DataResultModel](GO_URL+"/data/"+insertedData.DataId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestChainConsistency(t *testing.T) {

	readAllFromNodeApi, _ := utils.UrlRequest[[]chain.ChainModel](NODE_URL+"/chain", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]chain.ChainModel](GO_URL+"/chain", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[chain.ChainModel](NODE_URL+"/chain/"+insertedChain.ChainId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[chain.ChainModel](GO_URL+"/chain/"+insertedChain.ChainId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestApierrConsistency(t *testing.T) {
	readAllFromNodeApi, _ := utils.UrlRequest[[]apierr.ErrorModel](NODE_URL+"/error", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]apierr.ErrorModel](GO_URL+"/error", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[apierr.ErrorModel](NODE_URL+"/error/"+insertedError.ERROR_ID.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[apierr.ErrorModel](GO_URL+"/error/"+insertedError.ERROR_ID.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestAggregatorConsistency(t *testing.T) {
	readAllFromNodeApi, _ := utils.UrlRequest[[]aggregator.AggregatorResultModel](NODE_URL+"/aggregator?chain="+insertedChain.Name, "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]aggregator.AggregatorResultModel](GO_URL+"/aggregator?chain="+insertedChain.Name, "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[aggregator.AggregatorResultModel](NODE_URL+"/aggregator/"+insertedAggregator.AggregatorHash+"/"+insertedChain.Name, "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[aggregator.AggregatorResultModel](GO_URL+"/aggregator/"+insertedAggregator.AggregatorHash+"/"+insertedChain.Name, "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestAggregateConsistency(t *testing.T) {
	readAllFromNodeApi, _ := utils.UrlRequest[[]aggregate.AggregateModel](NODE_URL+"/aggregate", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]aggregate.AggregateModel](GO_URL+"/aggregate", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[aggregate.AggregateModel](NODE_URL+"/aggregate/"+insertedAggregate.AggregateId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[aggregate.AggregateModel](GO_URL+"/aggregate/"+insertedAggregate.AggregateId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

func TestAdapterConsistency(t *testing.T) {
	readAllFromNodeApi, _ := utils.UrlRequest[[]adapter.AdapterModel](NODE_URL+"/adapter", "GET", nil)
	readAllFromGoApi, _ := utils.UrlRequest[[]adapter.AdapterModel](GO_URL+"/adapter", "GET", nil)
	assert.EqualValues(t, readAllFromNodeApi, readAllFromGoApi)

	readSingleFromNodeApi, _ := utils.UrlRequest[adapter.AdapterModel](NODE_URL+"/adapter/"+insertedAdapter.AdapterId.String(), "GET", nil)
	readSingleFromGoApi, _ := utils.UrlRequest[adapter.AdapterModel](GO_URL+"/adapter/"+insertedAdapter.AdapterId.String(), "GET", nil)
	assert.EqualValues(t, readSingleFromNodeApi, readSingleFromGoApi)
}

// insert all data to read before test
func beforeAll() {
	var err error
	// insert service
	serviceInsertData := service.ServiceInsertModel{Name: "test-service"}
	insertedService, err = utils.UrlRequest[service.ServiceModel](GO_URL+"/service", "POST", serviceInsertData)
	if err != nil {
		panic("failed to insert service")
	}

	// insert chain
	chainInsertData := chain.ChainInsertModel{Name: "test-chain"}
	insertedChain, err = utils.UrlRequest[chain.ChainModel](GO_URL+"/chain", "POST", chainInsertData)
	if err != nil {
		panic("failed to insert chain")
	}

	// insert adapter & feed
	_adapterInsertData := _AdapterInsertModel{
		ADAPTER_HASH: "0xbb555a249d01133784fa04c608ce03c129f73f2a1ef7473d0cfffdc4bcba794e",
		NAME:         "BTC-USD",
		DECIMALS:     8,
	}

	adapterInsertData := AdapterInsertModel{
		_AdapterInsertModel: _adapterInsertData,
		FEEDS: []feed.FeedInsertModel{
			{
				Name: "Binance-BTC-USD-adapter",
				Definition: json.RawMessage(`{
					"url": "https://api.binance.us/api/v3/ticker/price?symbol=BTCUSD",
					"headers": {
						"Content-Type": "application/json"
					},
					"method": "GET",
					"reducers": [
						{
							"function": "PARSE",
							"args": [
								"price"
							]
						},
						{
							"function": "POW10",
							"args": 8
						},
						{
							"function": "ROUND"
						}
					]
				}`),
			},
		},
	}
	insertedAdapter, err = utils.UrlRequest[adapter.AdapterModel](GO_URL+"/adapter", "POST", adapterInsertData)
	if err != nil {
		panic("failed to insert adapter")
	}
	insertedFeeds, err = utils.UrlRequest[[]feed.FeedModel](GO_URL+"/feed/adapter/"+insertedAdapter.AdapterId.String(), "GET", nil)
	if err != nil {
		panic("failed to get inserted feeds")
	}

	// insert aggregator
	// aggregatorInsertData := aggregator.AggregatorInsertModel{
	// 	AGGREGATOR_HASH:    "0x9ca45583d7b9b061d9e8a20d6a874fcfba50c7a9dbc9c65c3792b4ef0b31e7b9",
	// 	ACTIVE:             false,
	// 	NAME:               "BTC-USD",
	// 	ADDRESS:            "0x222",
	// 	HEARTBEAT:          10000,
	// 	THRESHOLD:          0.04,
	// 	ABSOLUTE_THRESHOLD: 0.1,
	// 	ADAPTER_HASH:       insertedAdapter.ADAPTER_HASH,
	// 	CHAIN:              insertedChain.NAME,
	// 	FETCHER_TYPE:       0,
	// }
	aggregatorInsertData := map[string]interface{}{
		"aggregatorHash":    "0x9ca45583d7b9b061d9e8a20d6a874fcfba50c7a9dbc9c65c3792b4ef0b31e7b9",
		"active":            false,
		"name":              "BTC-USD",
		"address":           "0x222",
		"heartbeat":         10000,
		"threshold":         0.04,
		"absoluteThreshold": 0.1,
		"adapterHash":       _adapterInsertData.ADAPTER_HASH,
		"chain":             insertedChain.Name,
		"fetcherType":       0,
	}

	insertedAggregator, err = utils.UrlRequest[aggregator.AggregatorResultModel](GO_URL+"/aggregator", "POST", aggregatorInsertData)
	if err != nil {
		panic("failed to insert aggregator")
	}

	// insert error
	errInsertData := apierr.ErrorInsertModel{
		RequestId: "66649924661314489704239946349158829048302840686075232939396730072454733114998",
		Timestamp: &utils.CustomDateTime{Time: time.Now()},
		Code:      "10020",
		Name:      "MissingKeyInJson",
		Stack: `MissingKeyInJson
		at wrapper (file:///app/dist/worker/reducer.js:19:23)
		at file:///app/dist/utils.js:11:61
		at Array.reduce (<anonymous>)
		at file:///app/dist/utils.js:11:44
		at processRequest (file:///app/dist/worker/request-response.js:58:34)
		at process.processTicksAndRejections (node:internal/process/task_queues:95:5)
		at async Worker.wrapper [as processFn] (file:///app/dist/worker/request-response.js:27:25)
		at async Worker.processJob (/app/node_modules/bullmq/dist/cjs/classes/worker.js:339:28)
		at async Worker.retryIfFailed (/app/node_modules/bullmq/dist/cjs/classes/worker.js:513:24)`,
	}
	insertedError, err = utils.UrlRequest[apierr.ErrorModel](GO_URL+"/error", "POST", errInsertData)
	if err != nil {
		panic("failed to insert error")
	}

	// insert data
	// dataInsertData := data.BulkInsertModel{
	// 	DATA: []data.DataInsertModel{
	// 		{
	// 			AGGREGATOR_ID: insertedAggregator.AGGREGATOR_ID,
	// 			TIMESTAMP:     &utils.CustomDateTime{Time: time.Now()},
	// 			VALUE:         2241772466578,
	// 			FEED_ID:       insertedFeeds[0].FEED_ID,
	// 		},
	// 	},
	// }

	dataInsertData := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"aggregatorId": insertedAggregator.AggregatorId,
				"timestamp":    time.Now(),
				"value":        2241772466578,
				"feedId":       insertedFeeds[0].FeedId,
			},
		},
	}
	_, err = utils.UrlRequest[struct {
		COUNT int `json:"count"`
	}](GO_URL+"/data", "POST", dataInsertData)
	if err != nil {
		panic("failed to insert data")
	}
	insertedDataList, err := utils.UrlRequest[[]data.DataResultModel](GO_URL+"/data/feed/"+insertedFeeds[0].FeedId.String(), "GET", nil)
	if err != nil {
		panic("failed to read inserted data list")
	}

	insertedData = insertedDataList[0]

	// insert aggregate
	insertValue := utils.CustomInt64(10)
	aggregateInsertData := aggregate.AggregateInsertModel{
		Timestamp:    &utils.CustomDateTime{Time: time.Now()},
		AggregatorId: insertedAggregator.AggregatorId,
		Value:        &insertValue,
	}
	wrappedAggregateInsertData := aggregate.WrappedInsertModel{Data: aggregateInsertData}
	insertedAggregate, err = utils.UrlRequest[aggregate.AggregateModel](GO_URL+"/aggregate", "POST", wrappedAggregateInsertData)
	if err != nil {
		panic("failed to insert aggregate")
	}

	// insert listener
	listenerInsertData := listener.ListenerInsertModel{
		Address:   "0xa",
		EventName: "new_round(uint, uint80)",
		Chain:     insertedChain.Name,
		Service:   insertedService.Name,
	}
	insertedListener, err = utils.UrlRequest[listener.ListenerModel](GO_URL+"/listener", "POST", listenerInsertData)
	if err != nil {
		panic("failed to insert listener")
	}

	var portNumber = utils.CustomInt32(5000)
	// insert proxy
	proxyInsertData := proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     &portNumber,
	}
	insertedProxy, err = utils.UrlRequest[proxy.ProxyModel](GO_URL+"/proxy", "POST", proxyInsertData)
	if err != nil {
		panic("failed to insert proxy")
	}

	// insert reporter
	reporterInsertData := reporter.ReporterInsertModel{
		Address:       "0xa",
		PrivateKey:    "0xb",
		OracleAddress: "0xc",
		Chain:         insertedChain.Name,
		Service:       insertedService.Name,
	}
	insertedReporter, err = utils.UrlRequest[reporter.ReporterModel](GO_URL+"/reporter", "POST", reporterInsertData)
	if err != nil {
		panic("failed to insert reporter")
	}

	// insert vrf
	vrfInsertData := vrf.VrfInsertModel{
		Sk:      "ebeb5229570725793797e30a426d7ef8aca79d38ff330d7d1f28485d2366de32",
		Pk:      "045b8175cfb6e7d479682a50b19241671906f706bd71e30d7e80fd5ff522c41bf0588735865a5faa121c3801b0b0581440bdde24b03dc4c4541df9555d15223e82",
		PkX:     "41389205596727393921445837404963099032198113370266717620546075917307049417712",
		PkY:     "40042424443779217635966540867474786311411229770852010943594459290130507251330",
		KeyHash: "0x6f32373625e3d1f8f303196cbb78020ac2503acd1129e44b36b425781a9664ac",
		Chain:   insertedChain.Name,
	}
	insertedVrf, err = utils.UrlRequest[vrf.VrfModel](GO_URL+"/vrf", "POST", vrfInsertData)
	if err != nil {
		panic("failed to insert data")
	}
}

// remove all inserted data
func cleanup() {
	_, err := utils.UrlRequest[vrf.VrfModel](GO_URL+"/vrf/"+insertedVrf.VrfKeyId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete vrf")
	}

	_, err = utils.UrlRequest[reporter.ReporterModel](GO_URL+"/reporter/"+insertedReporter.ReporterId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete reporter")
	}

	_, err = utils.UrlRequest[proxy.ProxyModel](GO_URL+"/proxy/"+insertedProxy.Id.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete proxy")
	}

	_, err = utils.UrlRequest[listener.ListenerModel](GO_URL+"/listener/"+insertedListener.ListenerId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete proxy")
	}

	_, err = utils.UrlRequest[aggregate.AggregateModel](GO_URL+"/aggregate/"+insertedAggregate.AggregateId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete aggregate")
	}

	_, err = utils.UrlRequest[data.DataResultModel](GO_URL+"/data/"+insertedData.DataId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete data")
	}

	_, err = utils.UrlRequest[apierr.ErrorModel](GO_URL+"/error/"+insertedError.ERROR_ID.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete error")
	}

	_, err = utils.UrlRequest[aggregator.AggregatorResultModel](GO_URL+"/aggregator/"+insertedAggregator.AggregatorId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete aggregator")
	}

	_, err = utils.UrlRequest[feed.FeedModel](GO_URL+"/feed/"+insertedFeeds[0].FeedId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete feed")
	}

	_, err = utils.UrlRequest[adapter.AdapterModel](GO_URL+"/adapter/"+insertedAdapter.AdapterId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete adapter")
	}

	_, err = utils.UrlRequest[chain.ChainModel](GO_URL+"/chain/"+insertedChain.ChainId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete chain")
	}

	_, err = utils.UrlRequest[service.ServiceModel](GO_URL+"/service/"+insertedService.ServiceId.String(), "DELETE", nil)
	if err != nil {
		panic("failed to delete service")
	}
}
