package main

import (
	"math/big"

	"github.com/openrelayxyz/plugeth-utils/core"
	"github.com/openrelayxyz/plugeth-utils/restricted/types"
)

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// GetRewards calculates the mining reward.
// The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also calculated.
func GetRewards(config *PluginConfigurator, header *types.Header, uncles []*types.Header) (*big.Int, []*big.Int) {
	if config.IsEnabled(config.GetEthashECIP1017Transition, header.Number) {
		return ecip1017BlockReward(config, header, uncles)
	}

	blockReward := EthashBlockReward(config, header.Number)

	// Accumulate the rewards for the miner and any included uncles
	uncleRewards := make([]*big.Int, len(uncles))
	reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for i, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)

		ur := new(big.Int).Set(r)
		uncleRewards[i] = ur

		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}

	return reward, uncleRewards
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The coinbase of each uncle block is also rewarded.
func AccumulateRewards(config *PluginConfigurator, state core.RWStateDB, header *types.Header, uncles []*types.Header) {
	minerReward, uncleRewards := GetRewards(config, header, uncles)
	for i, uncle := range uncles {
		state.AddBalance(uncle.Coinbase, uncleRewards[i])
	}
	state.AddBalance(header.Coinbase, minerReward)
}

// As of "Era 2" (zero-index era 1), uncle miners and winners are rewarded equally for each included block.
// So they share this function.
func getEraUncleBlockReward(era *big.Int, blockReward *big.Int) *big.Int {
	return new(big.Int).Div(GetBlockWinnerRewardByEra(era, blockReward), big32)
}

// GetBlockUncleRewardByEra gets called _for each uncle miner_ associated with a winner block's uncles.
func GetBlockUncleRewardByEra(era *big.Int, header, uncle *types.Header, blockReward *big.Int) *big.Int {
	// Era 1 (index 0):
	//   An extra reward to the winning miner for including uncles as part of the block, in the form of an extra 1/32 (0.15625ETC) per uncle included, up to a maximum of two (2) uncles.
	if era.Cmp(big.NewInt(0)) == 0 {
		r := new(big.Int)
		r.Add(uncle.Number, big8) // 2,534,998 + 8              = 2,535,006
		r.Sub(r, header.Number)   // 2,535,006 - 2,534,999        = 7
		r.Mul(r, blockReward)     // 7 * 5e+18               = 35e+18
		r.Div(r, big8)            // 35e+18 / 8                            = 7/8 * 5e+18

		return r
	}
	return getEraUncleBlockReward(era, blockReward)
}

// GetBlockWinnerRewardForUnclesByEra gets called _per winner_, and accumulates rewards for each included uncle.
// Assumes uncles have been validated and limited (@ func (v *BlockValidator) VerifyUncles).
func GetBlockWinnerRewardForUnclesByEra(era *big.Int, uncles []*types.Header, blockReward *big.Int) *big.Int {
	r := big.NewInt(0)

	for range uncles {
		r.Add(r, getEraUncleBlockReward(era, blockReward)) // can reuse this, since 1/32 for winner's uncles remain unchanged from "Era 1"
	}
	return r
}

// GetRewardByEra gets a block reward at disinflation rate.
// Constants MaxBlockReward, DisinflationRateQuotient, and DisinflationRateDivisor assumed.
func GetBlockWinnerRewardByEra(era *big.Int, blockReward *big.Int) *big.Int {
	if era.Cmp(big.NewInt(0)) == 0 {
		return new(big.Int).Set(blockReward)
	}

	// MaxBlockReward _r_ * (4/5)**era == MaxBlockReward * (4**era) / (5**era)
	// since (q/d)**n == q**n / d**n
	// qed
	var q, d, r *big.Int = new(big.Int), new(big.Int), new(big.Int)

	q.Exp(DisinflationRateQuotient, era, nil)
	d.Exp(DisinflationRateDivisor, era, nil)

	r.Mul(blockReward, q)
	r.Div(r, d)

	return r
}

func ecip1017BlockReward(config *PluginConfigurator, header *types.Header, uncles []*types.Header) (*big.Int, []*big.Int) {
	blockReward := FrontierBlockReward

	// Ensure value 'era' is configured.
	eraLen := config.GetEthashECIP1017EraRounds()
	era := GetBlockEra(header.Number, new(big.Int).SetUint64(*eraLen))
	wr := GetBlockWinnerRewardByEra(era, blockReward)                    // wr "winner reward". 5, 4, 3.2, 2.56, ...
	wurs := GetBlockWinnerRewardForUnclesByEra(era, uncles, blockReward) // wurs "winner uncle rewards"
	wr.Add(wr, wurs)

	// Reward uncle miners.
	uncleRewards := make([]*big.Int, len(uncles))
	for i, uncle := range uncles {
		ur := GetBlockUncleRewardByEra(era, header, uncle, blockReward)
		uncleRewards[i] = ur
	}

	return wr, uncleRewards
}

// GetBlockEra gets which "Era" a given block is within, given an era length (ecip-1017 has era=5,000,000 blocks)
// Returns a zero-index era number, so "Era 1": 0, "Era 2": 1, "Era 3": 2 ...
func GetBlockEra(blockNum, eraLength *big.Int) *big.Int {
	// If genesis block or impossible negative-numbered block, return zero-val.
	if blockNum.Sign() < 1 {
		return new(big.Int)
	}

	remainder := big.NewInt(0).Mod(big.NewInt(0).Sub(blockNum, big.NewInt(1)), eraLength)
	base := big.NewInt(0).Sub(blockNum, remainder)

	d := big.NewInt(0).Div(base, eraLength)
	dremainder := big.NewInt(0).Mod(d, big.NewInt(1))

	return new(big.Int).Sub(d, dremainder)
}

func EthashBlockReward(c *PluginConfigurator, n *big.Int) *big.Int {
	// Select the correct block reward based on chain progression
	blockReward := FrontierBlockReward
	if c == nil || n == nil {
		return blockReward
	}

	if c.IsEnabled(c.GetEthashEIP1234Transition, n) {
		return EIP1234FBlockReward
	} else if c.IsEnabled(c.GetEthashEIP649Transition, n) {
		return EIP649FBlockReward
	} else if len(c.GetEthashBlockRewardSchedule()) > 0 {
		// Because the map is not necessarily sorted low-high, we
		// have to ensure that we're walking upwards only.
		var lastActivation uint64
		for activation, reward := range c.GetEthashBlockRewardSchedule() {
			if activation <= n.Uint64() { // Is forked
				if activation >= lastActivation {
					lastActivation = activation
					blockReward = reward
				}
			}
		}
	}

	return blockReward
}
