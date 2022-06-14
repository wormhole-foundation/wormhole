// This file is maintained by hand. Add / remove / update entries as appropriate.
package governor

func chainList() []chainConfigEntry {
	return []chainConfigEntry{
		chainConfigEntry{emitterChainID: 2, emitterAddr: "0x3ee18B2214AFF97000D974cf647E7C347E8fa585", dailyLimit: 1000000},
		chainConfigEntry{emitterChainID: 5, emitterAddr: "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE", dailyLimit: 1000000},
	}
}
