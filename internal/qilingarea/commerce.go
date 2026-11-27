package qilingarea

import (
	"errors"
	"strconv"
	"strings"

	pb "google.golang.org/protobuf/proto"
	gamepb "tytd_go/internal/proto"
)

// MallPurchaseKind describes a server-configured activity mall row. It is a
// classification only; callers must still honor the game's purchase flow.
type MallPurchaseKind int

const (
	MallUnsupported MallPurchaseKind = iota
	MallFree
	MallAdGated
	MallJade
)

const mallGoodsCategory = int32(0) // Client MailConfig.EMallBuyType.Goods.

// MarshalMallPurchaseRequest builds a single goods purchase request. Callers
// must first enforce the category/price/ad policy; this helper never performs
// network I/O and leaves isUseADTime false.
func MarshalMallPurchaseRequest(activityID, mallID int32) ([]byte, error) {
	if activityID <= 0 || mallID <= 0 {
		return nil, errors.New("activityId and mallId must be positive")
	}
	return pb.Marshal(&gamepb.ReqBuyActivityGoods{ActivityId: activityID, MallId: mallID, Count: 1})
}

// MarshalRechargeClaimRequest builds the client's per-condition 21004 request.
// It has no side effects until a transport sends the returned bytes.
func MarshalRechargeClaimRequest(activityID, conditionID int32) ([]byte, error) {
	if activityID <= 0 || conditionID <= 0 {
		return nil, errors.New("activityId and conditionId must be positive")
	}
	return pb.Marshal(&gamepb.ReqGetActivityConditionReward{ActivityId: pb.Int32(activityID), ConditionId: pb.Int32(conditionID)})
}

// ClassifyMallPurchase follows the existing event-shop policy: free rows,
// server-gated ad rows, and jade-cost rows may be planned. Other currencies
// fail closed. The server enforces ad cooldown/eligibility on 21005.
func ClassifyMallPurchase(row *gamepb.ActivityMallTempMsg, jadeItemID int32) MallPurchaseKind {
	if row == nil || row.GetCategory() != mallGoodsCategory {
		return MallUnsupported
	}
	price := strings.TrimSpace(row.GetPrice())
	ad := strings.HasPrefix(strings.TrimSpace(row.GetParam()), "3|")
	if price == "0" || strings.HasSuffix(price, "=0") {
		if ad {
			return MallAdGated
		}
		return MallFree
	}
	if jadeItemID <= 0 {
		return MallUnsupported
	}
	parts := strings.SplitN(price, "=", 2)
	if len(parts) != 2 {
		return MallUnsupported
	}
	itemID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil || int32(itemID) != jadeItemID {
		return MallUnsupported
	}
	amount, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || amount <= 0 {
		return MallUnsupported
	}
	if ad {
		return MallAdGated
	}
	return MallJade
}

// PlanMallPurchases returns free, ad-gated (server-cooldown), and jade-cost
// IDs up to server-provided buy limits. The caller sends one 21005 at a time;
// server refusals such as an active ad cooldown stop that row for this sweep.
func PlanMallPurchases(rows []*gamepb.ActivityMallTempMsg, bought map[int32]int64, jadeItemID int32) []int32 {
	var out []int32
	for _, row := range rows {
		kind := ClassifyMallPurchase(row, jadeItemID)
		if kind != MallFree && kind != MallAdGated && kind != MallJade {
			continue
		}
		id := row.GetId()
		if id <= 0 {
			continue
		}
		limit := row.GetBuyLimit()
		if limit < 1 { // Existing event-shop policy caps an absent limit to one.
			limit = 1
		}
		if int64(limit)-bought[id] > 0 {
			out = append(out, id)
		}
	}
	return out
}

// EligibleRechargeConditions selects only unclaimed type-84 milestones whose
// current server progress meets the configured threshold. The actual claim is
// a separate explicit online operation.
func EligibleRechargeConditions(configs []*gamepb.ActivityConditionConfig, data []*gamepb.ActivityConditionData) []int32 {
	progress := make(map[int32]*gamepb.ActivityConditionData, len(data))
	for _, row := range data {
		if row != nil {
			progress[row.GetConditionId()] = row
		}
	}
	var out []int32
	for _, cfg := range configs {
		if cfg == nil || cfg.GetType() != 84 || cfg.GetConditionId() <= 0 {
			continue
		}
		status := progress[cfg.GetConditionId()]
		if status == nil || status.GetIsGetReward() {
			continue
		}
		params := cfg.GetParam()
		if len(params) == 0 {
			continue
		}
		threshold, err := strconv.ParseInt(strings.TrimSpace(params[0]), 10, 64)
		if err != nil || threshold <= 0 {
			continue
		}
		value, err := strconv.ParseInt(strings.TrimSpace(status.GetValue()), 10, 64)
		if err == nil && value >= threshold {
			out = append(out, cfg.GetConditionId())
		}
	}
	return out
}
