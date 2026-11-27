package tytd_go

import (
	"testing"

	"google.golang.org/protobuf/proto"
	gamepb "tytd_go/internal/proto"
)

func TestQilingCommerceMergeUsesOnlyType131ServerDetail(t *testing.T) {
	p := &EventPlayer{}
	makeDetail := func(activityType int32) []byte {
		rsp := &gamepb.RspGetActivityDetail{
			Ret: 0,
			Activity: &gamepb.ActivityCommonData{
				ActivityId:        proto.Int32(9064650),
				MallBuyCountList:  []*gamepb.ActivityMallBuyCountData{{MallId: proto.Int32(77), Count: proto.Int64(1)}},
				ConditionDataList: []*gamepb.ActivityConditionData{{ConditionId: proto.Int32(84), Value: proto.String("100"), IsGetReward: proto.Bool(true)}},
				DetailConfig: &gamepb.ActivityDetailConfig{CommonConfig: &gamepb.ActivityCommonConfig{
					MainConfig:      &gamepb.ActivityMainConfig{ActivityId: proto.Int32(9064650), Type: proto.Int32(activityType)},
					MallConfig:      []*gamepb.ActivityMallConfig{{MallTempMsg: &gamepb.ActivityMallTempMsg{Id: proto.Int32(77), Name: proto.String("Gói Tiên Ngọc"), Price: proto.String("100000=60"), BuyLimit: proto.Int32(2)}}},
					ConditionConfig: []*gamepb.ActivityConditionConfig{{ConditionId: proto.Int32(84), Type: proto.Int32(84), Param: []string{"100"}}},
				}},
			},
		}
		data, err := proto.Marshal(rsp)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	p.qlMergeCommerceConfig(makeDetail(qilingAreaType))
	p.RbtMu.Lock()
	if len(p.qlMallRows) != 1 || p.qlMallBought[77] != 1 || len(p.qlConditionConfig) != 1 || len(p.qlConditionData) != 1 {
		p.RbtMu.Unlock()
		t.Fatalf("Type 131 detail not folded: mall=%d bought=%v conditions=%d progress=%d", len(p.qlMallRows), p.qlMallBought, len(p.qlConditionConfig), len(p.qlConditionData))
	}
	if !p.qlClaimed[84] {
		p.RbtMu.Unlock()
		t.Fatal("server-claimed milestone was not cached")
	}
	p.RbtMu.Unlock()

	previous := p.qlMallRows[0]
	p.qlMergeCommerceConfig(makeDetail(172))
	p.RbtMu.Lock()
	defer p.RbtMu.Unlock()
	if p.qlMallRows[0] != previous {
		t.Fatal("non-Type-131 detail replaced Qiling commerce config")
	}
}
