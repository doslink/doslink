package consensus

import (
	"strings"

	"github.com/doslink/doslink/protocol/bc"
)

//consensus variables
const (
	// Max gas that one block contains
	MaxBlockGas      = uint64(10000000)
	VMGasRate        = int64(200)
	StorageGasRate   = int64(1)
	MaxGasAmount     = int64(5000000)
	DefaultGasCredit = int64(30000)

	//config parameter for coinbase reward
	CoinbasePendingBlockNumber = uint64(10)
	subsidyReductionInterval   = ^uint64(0) // 2^64 - 1
	baseSubsidy                = uint64(750000000)
	InitialBlockSubsidy        = uint64(140700000750000000)

	// config for pow mining
	BlocksPerRetarget     = uint64(11)
	TargetSecondsPerBlock = uint64(13)
	SeedPerRetarget       = uint64(7)

	// MaxTimeOffsetSeconds is the maximum number of seconds a block time is allowed to be ahead of the current time
	MaxTimeOffsetSeconds = uint64(60 * 60)
	MedianTimeBlocks     = 11

	PayToWitnessScriptHashDataSize = 20
	CoinbaseArbitrarySizeLimit     = 128

	NativeAssetAlias = "DOS"
	NativeChainName  = "Doslink"
)

// NativeAssetID is NativeAsset's asset id, the soul asset of the Chain
// ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
var NativeAssetID = &bc.AssetID{
	V0: uint64(18446744073709551615),
	V1: uint64(18446744073709551615),
	V2: uint64(18446744073709551615),
	V3: uint64(18446744073709551615),
}

// InitialSeed is SHA3-256 of Byte[0^32]
var InitialSeed = &bc.Hash{
	V0: uint64(1731360381494106119),
	V1: uint64(2456720262922970904),
	V2: uint64(438282961664083713),
	V3: uint64(1303800995994077458),
}

// NativeAssetDefinitionMap is the ....
var NativeAssetDefinitionMap = map[string]interface{}{
	"name":        NativeAssetAlias,
	"symbol":      NativeAssetAlias,
	"decimals":    8,
	"description": strings.Title(NativeChainName) + ` Official Issue`,
}

// BlockSubsidy calculate the coinbase rewards on given block height
func BlockSubsidy(height uint64) uint64 {
	if height == 0 {
		return InitialBlockSubsidy
	}
	return baseSubsidy >> uint(height/subsidyReductionInterval)
}

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
type Checkpoint struct {
	Height uint64
	Hash   bc.Hash
}

// Params store the config for different network
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name        string
	Checkpoints []Checkpoint
}

// ActiveNetParams is ...
var ActiveNetParams = MainNetParams

// NetParams is the correspondence between chain_id and Params
var NetParams = map[string]Params{
	"mainnet": MainNetParams,
	"testnet": TestNetParams,
	"solonet": SoloNetParams,
}

// MainNetParams is the config for production
var MainNetParams = Params{
	Name:        "main",
	Checkpoints: []Checkpoint{
	},
}

// TestNetParams is the config for test-net
var TestNetParams = Params{
	Name:        "test",
	Checkpoints: []Checkpoint{
	},
}

// SoloNetParams is the config for test-net
var SoloNetParams = Params{
	Name:        "solo",
	Checkpoints: []Checkpoint{},
}
