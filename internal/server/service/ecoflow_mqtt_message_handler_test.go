package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractIndicesAndValueList(t *testing.T) {
	input := map[string]interface{}{
		"ecolinker_emsBpAliveNum":                      2,
		"ecolinker_mpptHeartBeat_0_mpptPv_0_amp":       2,
		"ecolinker_mpptHeartBeat_0_mpptPv_0_pwr":       0,
		"ecolinker_mpptHeartBeat_0_mpptPv_0_vol":       52,
		"ecolinker_mpptHeartBeat_0_mpptPv_1_amp":       2323,
		"ecolinker_mpptHeartBeat_0_mpptPv_1_pwr":       323,
		"ecolinker_mpptHeartBeat_0_mpptPv_1_vol":       4,
		"ecolinker_mpptHeartBeat_1_mpptPv_1_vol":       42,
		"ecolinker_mpptHeartBeat_1_mpptPv_2_vol":       666,
		"ecolinker_mpptHeartBeat_0__1__2_mpptPv_2_abc": 1.6,
	}

	res := extractIndicesAndValueList(input)
	a := assert.New(t)
	a.Equal(len(input), len(res))
}
