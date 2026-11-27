package qilingarea

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
	gamepb "tytd_go/internal/proto"
)

func TestPlanMallPurchasesAllowsFreeAdGatedAndJadeButKeepsMoney(t *testing.T) {
	rows := []*gamepb.ActivityMallTempMsg{
		{Id: proto.Int32(1), Price: proto.String("0"), BuyLimit: proto.Int32(1)},
		{Id: proto.Int32(2), Price: proto.String("100000=60"), BuyLimit: proto.Int32(2)},
		{Id: proto.Int32(3), Price: proto.String("600"), Param: proto.String("1|0|0"), BuyLimit: proto.Int32(1)},
		{Id: proto.Int32(4), Price: proto.String("0"), Param: proto.String("3|1800|0|0"), BuyLimit: proto.Int32(3)},
		{Id: proto.Int32(5), Price: proto.String("99999=10"), BuyLimit: proto.Int32(1)},
		{Id: proto.Int32(6), Price: proto.String("0"), BuyLimit: proto.Int32(-1)},
		{Id: proto.Int32(7), Price: proto.String("100000=5"), Category: proto.Int32(1), BuyLimit: proto.Int32(2)},
	}
	got := PlanMallPurchases(rows, map[int32]int64{2: 1, 4: 1}, 100000)
	want := []int32{1, 2, 4, 6} // missing/negative server limit is capped to one like other event shops
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PlanMallPurchases()=%v want %v", got, want)
	}
	if ClassifyMallPurchase(rows[3], 100000) != MallAdGated {
		t.Fatal("ad-gated package was not recognized")
	}
	if ClassifyMallPurchase(rows[2], 100000) != MallUnsupported {
		t.Fatal("money-priced recharge package was not rejected")
	}
}

func TestEligibleRechargeConditionsRequiresProgressAndUnclaimedState(t *testing.T) {
	configs := []*gamepb.ActivityConditionConfig{
		{ActivityId: proto.Int32(9064650), ConditionId: proto.Int32(84), Type: proto.Int32(84), Param: []string{"100"}},
		{ActivityId: proto.Int32(9064650), ConditionId: proto.Int32(85), Type: proto.Int32(84), Param: []string{"500"}},
		{ActivityId: proto.Int32(9064650), ConditionId: proto.Int32(86), Type: proto.Int32(3), Param: []string{"1"}},
		{ActivityId: proto.Int32(9064650), ConditionId: proto.Int32(87), Type: proto.Int32(84)},
	}
	data := []*gamepb.ActivityConditionData{
		{ConditionId: proto.Int32(84), Value: proto.String("100"), IsGetReward: proto.Bool(false)},
		{ConditionId: proto.Int32(85), Value: proto.String("999"), IsGetReward: proto.Bool(true)},
		{ConditionId: proto.Int32(86), Value: proto.String("99"), IsGetReward: proto.Bool(false)},
		{ConditionId: proto.Int32(87), Value: proto.String("1000"), IsGetReward: proto.Bool(false)},
	}
	got := EligibleRechargeConditions(configs, data)
	if !reflect.DeepEqual(got, []int32{84}) {
		t.Fatalf("EligibleRechargeConditions()=%v want [84]", got)
	}
}

func TestCommerceRequestBuildersEncodeOnlyOneGoodsUnitAndSelectedCondition(t *testing.T) {
	shopBody, err := MarshalMallPurchaseRequest(9064650, 77)
	if err != nil {
		t.Fatal(err)
	}
	var shop gamepb.ReqBuyActivityGoods
	if err := proto.Unmarshal(shopBody, &shop); err != nil {
		t.Fatal(err)
	}
	if shop.GetActivityId() != 9064650 || shop.GetMallId() != 77 || shop.GetCount() != 1 || shop.GetIsUseADTime() {
		t.Fatalf("unsafe/unexpected shop request: activity=%d mall=%d count=%d ad=%v", shop.GetActivityId(), shop.GetMallId(), shop.GetCount(), shop.GetIsUseADTime())
	}
	if _, err := MarshalMallPurchaseRequest(0, 77); err == nil {
		t.Fatal("invalid activity id accepted")
	}

	claimBody, err := MarshalRechargeClaimRequest(9064650, 84)
	if err != nil {
		t.Fatal(err)
	}
	var claim gamepb.ReqGetActivityConditionReward
	if err := proto.Unmarshal(claimBody, &claim); err != nil {
		t.Fatal(err)
	}
	if claim.GetActivityId() != 9064650 || claim.GetConditionId() != 84 {
		t.Fatalf("wrong milestone request: activity=%d condition=%d", claim.GetActivityId(), claim.GetConditionId())
	}
	if _, err := MarshalRechargeClaimRequest(9064650, 0); err == nil {
		t.Fatal("invalid condition id accepted")
	}
}
