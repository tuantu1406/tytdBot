package tytd_go

// Ngự Khôi Tranh Phong event module: verified wire builders, conservative
// shop planner, and the shared non-purchase cycle actions. Player lifecycle
// adapters remain in player_ngockhoi.go and player_event_ngockhoi.go.

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	googleproto "google.golang.org/protobuf/proto"
	"tytd_go/internal/pack"
	gameproto "tytd_go/internal/proto"
)

const (
	ngocKhoiActivityType        int32 = 321
	ngocKhoiGuildConditionID    int32 = 10003
	ngocKhoiGuildConditionIDs         = "10001,10002,10003,10004,10005,10006,10007,10008"
	ngocKhoiEnterCmd                  = "00005141"
	ngocKhoiTreeDonateCmd             = "000397ff"
	ngocKhoiTreeDonateResp            = "7d83"
	ngocKhoiTreeClaimCmd              = "000397fe"
	ngocKhoiTreeClaimResp             = "7d82"
	ngocKhoiTreeLevelRewardCmd        = "000397fd"
	ngocKhoiTreeLevelRewardResp       = "7d81"
	ngocKhoiEquipReorderCmd           = "000397f4"
	ngocKhoiEquipReorderResp          = "7d74"
)

func ngocKhoiEnterPayload(mode, activityID int32) []byte {
	return ngocKhoiFields(321, mode, activityID)
}
func ngocKhoiTreeDonatePayload(activityID, itemID, count int32) []byte {
	return ngocKhoiFields(activityID, itemID, count)
}
func ngocKhoiTreeClaimPayload(activityID, claimMode int32) []byte {
	return ngocKhoiFields(activityID, claimMode)
}
func ngocKhoiTreeLevelRewardPayload(activityID int32) []byte { return ngocKhoiFields(activityID) }
func ngocKhoiEquipReorderPayload(activityID, mode, from, to, slot int32) []byte {
	return ngocKhoiFields(activityID, mode, from, to, slot)
}
func ngocKhoiFields(values ...int32) []byte {
	b := make([]byte, 0, len(values)*3)
	for i, v := range values {
		b = protowire.AppendTag(b, protowire.Number(i+1), protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(v))
	}
	return b
}

type NgocKhoiShopKind uint8

const (
	NgocKhoiShopFree NgocKhoiShopKind = iota + 1
	NgocKhoiShopAd
	NgocKhoiShopTienNgoc
)

type NgocKhoiShopItem struct {
	ActivityID    int32
	MallID        int32
	Name          string
	Kind          NgocKhoiShopKind
	Limit, Bought int64
}

func ngocKhoiShopItems(activityID int32, rows []*gameproto.ActivityMallTempMsg, bought map[int32]int64) []NgocKhoiShopItem {
	items := make([]NgocKhoiShopItem, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.GetId() == 0 || row.GetBuyLimit() <= 0 {
			continue
		}
		kind, ok := ngocKhoiShopKind(row.GetPrice(), row.GetParam())
		if ok {
			items = append(items, NgocKhoiShopItem{activityID, row.GetId(), row.GetName(), kind, int64(row.GetBuyLimit()), bought[row.GetId()]})
		}
	}
	return items
}
func ngocKhoiShopKind(price, param string) (NgocKhoiShopKind, bool) {
	if p := strings.Split(param, "|"); len(p) > 0 && strings.TrimSpace(p[0]) == "3" {
		return NgocKhoiShopAd, true
	}
	if price == "0" || strings.HasSuffix(price, "=0") {
		return NgocKhoiShopFree, true
	}
	if strings.HasPrefix(price, "100000=") {
		if n, err := strconv.ParseInt(strings.TrimPrefix(price, "100000="), 10, 64); err == nil && n > 0 {
			return NgocKhoiShopTienNgoc, true
		}
	}
	return 0, false
}
func ngocKhoiShopRequests(activityID int32, rows []*gameproto.ActivityMallTempMsg, bought map[int32]int64) []*gameproto.ReqBuyActivityGoods {
	var out []*gameproto.ReqBuyActivityGoods
	for _, item := range ngocKhoiShopItems(activityID, rows, bought) {
		for n := item.Bought; n < item.Limit; n++ {
			out = append(out, &gameproto.ReqBuyActivityGoods{ActivityId: item.ActivityID, MallId: item.MallID, Count: 1, IsUseADTime: false})
		}
	}
	return out
}
func marshalNgocKhoiShopRequest(req *gameproto.ReqBuyActivityGoods) ([]byte, error) {
	if req == nil || req.GetActivityId() == 0 || req.GetMallId() == 0 || req.GetCount() != 1 {
		return nil, fmt.Errorf("invalid Ngoc Khoi shop request")
	}
	return googleproto.Marshal(req)
}

func (p *Player) ngocKhoiVerifiedCycle(activityID int32, slot string) {
	p.NgocKhoiMu.Lock()
	if p.ngocKhoiActionSlot == slot {
		p.NgocKhoiMu.Unlock()
		return
	}
	p.ngocKhoiActionSlot = slot
	p.NgocKhoiMu.Unlock()
	p.write(pack.Pack(ngocKhoiEnterCmd, ngocKhoiEnterPayload(1, activityID), p.UID))
	p.write(pack.Pack(ngocKhoiTreeClaimCmd, ngocKhoiTreeClaimPayload(activityID, 0), p.UID))
	p.write(pack.Pack(ngocKhoiTreeLevelRewardCmd, ngocKhoiTreeLevelRewardPayload(activityID), p.UID))
	log.Printf("[%s] [NgocKhoi] đã gửi enter + nhận cây + nhận quà level cho phiên %s", p.Username, slot)
}
func (p *EventPlayer) ngocKhoiVerifiedCycle(activityID int32, slot string) {
	p.NgocKhoiMu.Lock()
	if p.ngocKhoiActionSlot == slot {
		p.NgocKhoiMu.Unlock()
		return
	}
	p.ngocKhoiActionSlot = slot
	p.NgocKhoiMu.Unlock()
	p.write(pack.Pack(ngocKhoiEnterCmd, ngocKhoiEnterPayload(1, activityID), p.UID))
	p.write(pack.Pack(ngocKhoiTreeClaimCmd, ngocKhoiTreeClaimPayload(activityID, 0), p.UID))
	p.write(pack.Pack(ngocKhoiTreeLevelRewardCmd, ngocKhoiTreeLevelRewardPayload(activityID), p.UID))
	log.Printf("[%s] [NgocKhoi] đã gửi enter + nhận cây + nhận quà level cho phiên %s", p.Username, slot)
}
