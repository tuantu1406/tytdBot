package tytd_go

import (
	"testing"

	"google.golang.org/protobuf/proto"

	gameproto "tytd_go/internal/proto"
)

func s(v string) *string { return &v }
func i32(v int32) *int32 { return &v }

func TestNgocKhoiShopOnlySelectsAuthorisedRows(t *testing.T) {
	rows := []*gameproto.ActivityMallTempMsg{
		{Id: i32(1), Name: s("free"), Price: s("0"), BuyLimit: i32(1)},
		{Id: i32(2), Name: s("ad"), Price: s("0"), Param: s("3|1800|0|0"), BuyLimit: i32(2)},
		{Id: i32(3), Name: s("jade"), Price: s("100000=388"), BuyLimit: i32(3)},
		{Id: i32(4), Name: s("money"), Price: s("1=225000"), Param: s("1|"), BuyLimit: i32(1)},
	}
	items := ngocKhoiShopItems(9064454, rows, map[int32]int64{2: 1, 3: 2})
	if len(items) != 3 {
		t.Fatalf("got %d authorised rows, want 3", len(items))
	}
	if items[0].Kind != NgocKhoiShopFree || items[1].Kind != NgocKhoiShopAd || items[2].Kind != NgocKhoiShopTienNgoc {
		t.Fatalf("unexpected kinds: %#v", items)
	}
	requests := ngocKhoiShopRequests(9064454, rows, map[int32]int64{2: 1, 3: 2})
	if len(requests) != 3 { // free 1 + ad 1 remaining + jade 1 remaining
		t.Fatalf("got %d requests, want 3", len(requests))
	}
	if requests[1].GetIsUseADTime() != true || requests[2].GetMallId() != 3 {
		t.Fatalf("request classification is wrong: %#v", requests)
	}
}

func TestNgocKhoiShopRejectsUnverifiedTienNgocAndMoney(t *testing.T) {
	rows := []*gameproto.ActivityMallTempMsg{
		{Id: i32(5), Price: s("388"), BuyLimit: i32(3)},
		{Id: i32(6), Price: s("1=129000"), Param: s("1|"), BuyLimit: i32(3)},
	}
	if got := ngocKhoiShopRequests(1, rows, nil); len(got) != 0 {
		t.Fatalf("unverified/money rows produced %d requests", len(got))
	}
}

func TestMarshalNgocKhoiShopRequest(t *testing.T) {
	req := &gameproto.ReqBuyActivityGoods{ActivityId: 9064454, MallId: 400000003, Count: 1}
	b, err := marshalNgocKhoiShopRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded gameproto.ReqBuyActivityGoods
	if err := proto.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetActivityId() != 9064454 || decoded.GetMallId() != 400000003 || decoded.GetCount() != 1 || decoded.GetIsUseADTime() {
		t.Fatalf("decoded request mismatch: %v", &decoded)
	}
}
