package tytd_go

import (
	"fmt"
	"strconv"
	"strings"

	googleproto "google.golang.org/protobuf/proto"

	gameproto "tytd_go/internal/proto"
)

// NgocKhoiShopKind is the small, explicitly authorised subset of the event
// shop. Money/recharge rows intentionally have no value in this enum.
type NgocKhoiShopKind uint8

const (
	NgocKhoiShopFree NgocKhoiShopKind = iota + 1
	NgocKhoiShopAd
	NgocKhoiShopTienNgoc
)

type NgocKhoiShopItem struct {
	ActivityID int32
	MallID     int32
	Name       string
	Kind       NgocKhoiShopKind
	Limit      int64
	Bought     int64
}

// ngocKhoiShopItems folds the verified 21003 mall rows into buyable items.
// It is deliberately conservative: a Tiên Ngọc row must advertise the
// known currency id 100000 in price (the same format used by existing event
// shops), while all recharge rows are rejected. A caller can pass the
// server's mallBuyCountList so the function remains idempotent for the day.
func ngocKhoiShopItems(activityID int32, rows []*gameproto.ActivityMallTempMsg, bought map[int32]int64) []NgocKhoiShopItem {
	items := make([]NgocKhoiShopItem, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.GetId() == 0 {
			continue
		}
		limit := int64(row.GetBuyLimit())
		if limit <= 0 {
			continue
		}
		kind, ok := ngocKhoiShopKind(row.GetPrice(), row.GetParam())
		if !ok {
			continue
		}
		items = append(items, NgocKhoiShopItem{
			ActivityID: activityID,
			MallID:     row.GetId(),
			Name:       row.GetName(),
			Kind:       kind,
			Limit:      limit,
			Bought:     bought[row.GetId()],
		})
	}
	return items
}

func ngocKhoiShopKind(price, param string) (NgocKhoiShopKind, bool) {
	price = strings.TrimSpace(price)
	parts := strings.Split(param, "|")
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "3" {
		return NgocKhoiShopAd, true
	}
	if price == "0" || strings.HasSuffix(price, "=0") {
		return NgocKhoiShopFree, true
	}
	if strings.HasPrefix(price, "100000=") {
		value := strings.TrimPrefix(price, "100000=")
		if n, err := strconv.ParseInt(value, 10, 64); err == nil && n > 0 {
			return NgocKhoiShopTienNgoc, true
		}
	}
	return 0, false
}

// ngocKhoiShopRequests creates at most one 21005 request per remaining unit.
// The caller must still serialize requests and handle the server response;
// this helper does not perform network I/O and never creates money requests.
func ngocKhoiShopRequests(activityID int32, rows []*gameproto.ActivityMallTempMsg, bought map[int32]int64) []*gameproto.ReqBuyActivityGoods {
	items := ngocKhoiShopItems(activityID, rows, bought)
	requests := make([]*gameproto.ReqBuyActivityGoods, 0)
	for _, item := range items {
		remaining := item.Limit - item.Bought
		for i := int64(0); i < remaining; i++ {
			requests = append(requests, &gameproto.ReqBuyActivityGoods{
				ActivityId: item.ActivityID,
				MallId:     item.MallID,
				Count:      1,
				// Existing successful shop flows and the captured 21005 packet
				// send false here for every row. The server uses mall param to
				// enforce the ad gate; setting true makes the request invalid.
				IsUseADTime: false,
			})
		}
	}
	return requests
}

func marshalNgocKhoiShopRequest(req *gameproto.ReqBuyActivityGoods) ([]byte, error) {
	if req == nil || req.GetActivityId() == 0 || req.GetMallId() == 0 || req.GetCount() != 1 {
		return nil, fmt.Errorf("invalid Ngoc Khoi shop request")
	}
	return googleproto.Marshal(req)
}
