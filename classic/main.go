package main

import (
	"context"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/openrelayxyz/plugeth-utils/core"
	"github.com/openrelayxyz/plugeth-utils/restricted"
)

var (
	ClassicBootnodes = []string{

		"enode://942bf2f0754972391467765be1d98206926fc8ad0be8a49cd65e1730420c37fa63355bddb0ae5faa1d3505a2edcf8fad1cf00f3c179e244f047ec3a3ba5dacd7@176.9.51.216:30355", // @q9f ceibo
		"enode://0b0e09d6756b672ac6a8b70895da4fb25090b939578935d4a897497ffaa205e019e068e1ae24ac10d52fa9b8ddb82840d5d990534201a4ad859ee12cb5c91e82@176.9.51.216:30365", // @q9f ceibo
		"enode://b9e893ea9cb4537f4fed154233005ae61b441cd0ecd980136138c304fefac194c25a16b73dac05fc66a4198d0c15dd0f33af99b411882c68a019dfa6bb703b9d@18.130.93.66:30303",
	}

	dnsPrefixETC string = "enrtree://AJE62Q4DUX4QMMXEHCSSCSC65TDHZYSMONSD64P3WULVLSF6MRQ3K@"

	ClassicDNSNetwork1 string = dnsPrefixETC + "all.classic.blockd.info"

	snapDiscoveryURLs []string

	forkBlockIds = []uint64 {1150000, 2500000, 3000000, 5000000, 5900000, 8772000, 9573000, 10500839, 11700000, 13189133, 14525000, 19250000}

	forkTimeIds = []uint64{}
)

type ClassicService struct {
	backend core.Backend
	stack   core.Node
}

var (
	pl      core.PluginLoader
	backend restricted.Backend
	log     core.Logger
	events  core.Feed
)

var (
	httpApiFlagName = "http.api"
	mainnetFlag = "mainnet"
	goerliFlag = "goerli"
	sepoliaFlag = "sepolia"
	holeskyFlag = "holesky"

	networkPanicMsg = "This node is optimized to run the Ethereum Classic Network only, check datadir/plugins/ for a classic.so binary and remove it if this is not the desired behavior"
)

func Initialize(ctx core.Context, loader core.PluginLoader, logger core.Logger) { 
	pl = loader
	events = pl.GetFeed()
	log = logger
	v := ctx.String(httpApiFlagName)
	if v != "" {
		ctx.Set(httpApiFlagName, v+",plugeth")
	} else {
		ctx.Set(httpApiFlagName, "eth,net,web3,plugeth")
		
	}

	switch {
		case ctx.Bool(mainnetFlag):
			panic(networkPanicMsg)
		case ctx.Bool(goerliFlag):
			panic(networkPanicMsg)
		case ctx.Bool(sepoliaFlag):
			panic(networkPanicMsg)
		case ctx.Bool(holeskyFlag):
			panic(networkPanicMsg)
	}


	log.Info("Loaded Ethereum Classic plugin")
}

func Is1559(*big.Int) bool {
	return false
}

func Is160(num *big.Int) bool {
	r := num.Cmp(big.NewInt(3000000))
	return r >= 0
}

func IsShanghai(num *big.Int) bool {
	r := num.Cmp(big.NewInt(19250000))
	return r >= 0
}

func InitializeNode(node core.Node, backend restricted.Backend) {
	db := backend.ChainDb()

	cfg := []byte(`{
		"chainId": 61,
		"networkId": 1,
		"homesteadBlock": 1150000,
		"daoForkBlock": null,
		"daoForkSupport": false,
		"eip150Block": 2500000,
		"eip155Block": 3000000,
		"eip158Block": 8772000,
		"byzantiumBlock": 8772000,
		"constantinopleBlock": 9573000,
		"petersburgBlock": 9573000,
		"istanbulBlock": 10500839,
		"berlinBlock": 13189133,
		"londonBlock": 14525000,
		"ethash": {}
	}`)

	hash := core.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")

	if err := db.Put(append([]byte("ethereum-config-"), hash.Bytes()...), cfg); err != nil {
		log.Error("Error loading Classic config", "err", err)
	}
}

func GetAPIs(stack core.Node, backend core.Backend) []core.API {
	return []core.API{
		{
			Namespace: "plugeth",
			Version:   "1.0",
			Service:   &ClassicService{backend, stack},
			Public:    true,
		},
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &API{eHashForAPI},
			Public:    true,
		},
	}
}

// type API struct {
// 	ethash *Ethash
// }

func ForkIDs([]uint64, []uint64) ([]uint64, []uint64) {
	return forkBlockIds, forkTimeIds
}

func SetDefaultDataDir(path string) string {
	return filepath.Join(path, "classic")
}

func OpCodeSelect() []int {
	codes := []int{0x48}
	return codes
}

func SetNetworkId() *uint64 {
	var networkId *uint64
	classicNetworkId := uint64(1)
	networkId = &classicNetworkId
	return networkId 
}

func SetBootstrapNodes() []string {
	result := ClassicBootnodes
	return result
}

func SetETHDiscoveryURLs(lightSync bool) []string {

	url := ClassicDNSNetwork1
	if lightSync == true {
		url = strings.ReplaceAll(url, "all", "les")
	}
	result := []string{url}
	snapDiscoveryURLs = result

	return result
}

func SetSnapDiscoveryURLs() []string {
	return snapDiscoveryURLs
}

func (service *ClassicService) Test(ctx context.Context) string {
	return "total classic"
}
