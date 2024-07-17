package tests

import (
	"fmt"
	"testing"

	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/utils"
	"bisonai.com/orakl/api/vrf"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestVrf(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	var insertChain = chain.ChainInsertModel{Name: "vrf-test-chain"}

	var insertData = vrf.VrfInsertModel{
		Sk:      "ebeb5229570725793797e30a426d7ef8aca79d38ff330d7d1f28485d2366de32",
		Pk:      "045b8175cfb6e7d479682a50b19241671906f706bd71e30d7e80fd5ff522c41bf0588735865a5faa121c3801b0b0581440bdde24b03dc4c4541df9555d15223e82",
		PkX:     "41389205596727393921445837404963099032198113370266717620546075917307049417712",
		PkY:     "40042424443779217635966540867474786311411229770852010943594459290130507251330",
		KeyHash: "0x6f32373625e3d1f8f303196cbb78020ac2503acd1129e44b36b425781a9664ac",
		Chain:   "vrf-test-chain",
	}

	var updateData = vrf.VrfUpdateModel{
		Sk:      "ebeb5229570725793797e30a426d7ef8aca79d38ff330d7d1f28485d2366de32",
		Pk:      "045b8175cfb6e7d479682a50b19241671906f706bd71e30d7e80fd5ff522c41bf0588735865a5faa121c3801b0b0581440bdde24b03dc4c4541df9555d15223e82",
		PkX:     "41389205596727393921445837404963099032198113370266717620546075917307049417712",
		PkY:     "40042424443779217635966540867474786311411229770852010943594459290130507251330",
		KeyHash: "0x",
	}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")
	vrf.Routes(v1)
	chain.Routes(v1)

	// insert chain before test
	chainInsertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertChain)
	assert.Nil(t, err)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]vrf.VrfModel](app, "/api/v1/vrf", map[string]any{"chain": "vrf-test-chain"})
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	// insert
	insertResult, err := utils.PostRequest[vrf.VrfModel](app, "/api/v1/vrf", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]vrf.VrfModel](app, "/api/v1/vrf", map[string]any{"chain": "vrf-test-chain"})
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[vrf.VrfModel](app, "/api/v1/vrf/"+insertResult.VrfKeyId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should get inserted vrf")

	// patch
	patchResult, err := utils.PatchRequest[vrf.VrfModel](app, "/api/v1/vrf/"+insertResult.VrfKeyId.String(), updateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[vrf.VrfModel](app, "/api/v1/vrf/"+insertResult.VrfKeyId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, patchResult, "should be patched")

	// delete
	deleteResult, err := utils.DeleteRequest[vrf.VrfModel](app, "/api/v1/vrf/"+insertResult.VrfKeyId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	// read all after delete
	readAllResultAfterDeletion, err := utils.GetRequest[[]vrf.VrfModel](app, "/api/v1/vrf", map[string]any{"chain": "vrf-test-chain"})
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)

	// delete chain (cleanup)
	_, err = utils.DeleteRequest[chain.ChainModel](app, "/api/v1/chain/"+chainInsertResult.ChainId.String(), nil)
	assert.Nil(t, err)
}
