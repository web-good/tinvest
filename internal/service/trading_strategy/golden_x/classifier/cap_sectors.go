package classifier

import (
	"sort"

	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
)

// sectorCapPerTick is the maximum number of buy signals allowed per sector in a
// single Trade tick. Excess signals are moved to CappedBuyShares.
const sectorCapPerTick = 2

// CapSectors enforces sector concentration limits: at most sectorCapPerTick
// buy signals per sector pass through; the rest move to CappedBuyShares.
// Shares with empty Sector are never capped. SellShares are not affected.
func CapSectors(result gxmodel.TradeResult) gxmodel.TradeResult {
	// Group buy-share IDs by sector.
	sectorIDs := make(map[string][]string)
	for id, sr := range result.BuyShares {
		if sr.Sector == "" {
			continue // empty sector is never capped
		}
		sectorIDs[sr.Sector] = append(sectorIDs[sr.Sector], id)
	}

	for sector, ids := range sectorIDs {
		if len(ids) <= sectorCapPerTick {
			continue
		}

		// Sort by Score descending; use share ID as stable tiebreaker.
		sort.Slice(ids, func(i, j int) bool {
			si := result.BuyShares[ids[i]].Score
			sj := result.BuyShares[ids[j]].Score
			if si != sj {
				return si > sj
			}
			return ids[i] < ids[j]
		})

		_ = sector // used only as the map key above

		// Keep top-N, move the rest to CappedBuyShares.
		for _, id := range ids[sectorCapPerTick:] {
			result.CappedBuyShares[id] = result.BuyShares[id]
			delete(result.BuyShares, id)
		}
	}

	return result
}
