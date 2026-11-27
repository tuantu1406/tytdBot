package tytd_go

import (
	"fmt"
	"log"
	"strings"
	"time"
	"tytd_go/internal/proto"
	"tytd_go/internal/qilingarea"

	pb "google.golang.org/protobuf/proto"
)

const (
	qilingAreaType          = int32(131)
	qilingJadeItemID        = int32(100000)
	qilingMallGoodsCategory = int32(0) // Client MailConfig.EMallBuyType.Goods.
	qilingCommerceInterval  = 15 * time.Minute
)

// QilingArea commerce state is folded from server-authored 21003 responses;
// all fields are guarded by RbtMu because rbtRT already owns the response seat.
func (p *EventPlayer) qlMergeCommerceConfig(data []byte) {
	var rsp proto.RspGetActivityDetail
	if pb.Unmarshal(data, &rsp) != nil || rsp.GetRet() != 0 || rsp.GetActivity() == nil {
		return
	}
	activity := rsp.GetActivity()
	common := activity.GetDetailConfig().GetCommonConfig()
	if common == nil || common.GetMainConfig() == nil || common.GetMainConfig().GetType() != qilingAreaType {
		return
	}
	mallRows := make([]*proto.ActivityMallTempMsg, 0, len(common.GetMallConfig()))
	for _, row := range common.GetMallConfig() {
		if row != nil && row.GetMallTempMsg() != nil {
			mallRows = append(mallRows, row.GetMallTempMsg())
		}
	}
	bought := make(map[int32]int64, len(activity.GetMallBuyCountList()))
	for _, row := range activity.GetMallBuyCountList() {
		if row != nil {
			bought[row.GetMallId()] = row.GetCount()
		}
	}
	conditions := append([]*proto.ActivityConditionConfig(nil), common.GetConditionConfig()...)
	conditionData := make([]*proto.ActivityConditionData, 0, len(activity.GetConditionDataList()))
	for _, row := range activity.GetConditionDataList() {
		if row != nil {
			conditionData = append(conditionData, row)
		}
	}
	p.RbtMu.Lock()
	p.qlMallRows = mallRows
	p.qlMallBought = bought
	p.qlConditionConfig = conditions
	p.qlConditionData = conditionData
	p.qlClaimed = make(map[int32]bool)
	for _, row := range conditionData {
		if row.GetIsGetReward() {
			p.qlClaimed[row.GetConditionId()] = true
		}
	}
	p.RbtMu.Unlock()
}

// qilingCommerceLoop is an opt-out commerce helper for active Type 131 events.
// It follows rbtShopSweep's established 21005 policy: free/ad-cooldown rows and
// rows priced in jade are attempted one unit at a time; real-money rows are
// excluded. Ad eligibility/cooldown remains server-authoritative. Recharge
// milestones are claimed individually only when server state shows them ready.
func (p *EventPlayer) qilingCommerceLoop() {
	if !p.rbtSleepCtx(10 * time.Second) {
		return
	}
	var lastRun time.Time
	for {
		if p.OffQiling.Load() {
			if !p.rbtSleepCtx(time.Minute) {
				return
			}
			continue
		}
		var active *activityEntry
		for _, entry := range p.ActList.byType(qilingAreaType) {
			if entry.status(time.Now()) == "active" && (active == nil || entry.ActivityID > active.ActivityID) {
				e := entry
				active = &e
			}
		}
		if active == nil {
			if !p.rbtSleepCtx(5 * time.Minute) {
				return
			}
			continue
		}
		if lastRun.IsZero() || time.Since(lastRun) >= qilingCommerceInterval {
			p.qlCommerceSweep(active.ActivityID)
			lastRun = time.Now()
		}
		if !p.rbtSleepCtx(time.Minute) {
			return
		}
	}
}

func (p *EventPlayer) qlCommerceSweep(activityID int32) {
	body, err := pb.Marshal(&proto.ReqGetActivityDetail{ActivityId: activityID})
	if err != nil {
		return
	}
	if _, err = p.rbtRT(ACTIVITY_GET_DATA_CMD, SERVER_ACTIVITY_DETAIL, body); err != nil {
		log.Printf("[%s] [Qiling] không lấy được cấu hình cửa hàng/mốc nạp: %v", p.Username, err)
		return
	}
	p.qlBuyAllowedMallRows(activityID)
	p.qlClaimReadyRecharge(activityID)
}

func (p *EventPlayer) qlBuyAllowedMallRows(activityID int32) {
	p.RbtMu.Lock()
	rows := append([]*proto.ActivityMallTempMsg(nil), p.qlMallRows...)
	bought := make(map[int32]int64, len(p.qlMallBought))
	for id, count := range p.qlMallBought {
		bought[id] = count
	}
	p.RbtMu.Unlock()
	planned := qilingarea.PlanMallPurchases(rows, bought, qilingJadeItemID)
	byID := make(map[int32]*proto.ActivityMallTempMsg, len(rows))
	for _, row := range rows {
		if row != nil {
			byID[row.GetId()] = row
		}
	}
	for _, mallID := range planned {
		if p.OffQiling.Load() {
			return
		}
		row := byID[mallID]
		if row == nil {
			continue
		}
		kind := qilingarea.ClassifyMallPurchase(row, qilingJadeItemID)
		limit := row.GetBuyLimit()
		if limit < 1 { // Match other event shops: missing cap means one safe attempt.
			limit = 1
		}
		for bought[row.GetId()] < int64(limit) {
			body, err := qilingarea.MarshalMallPurchaseRequest(activityID, row.GetId())
			if err != nil {
				break
			}
			data, err := p.rbtRT(ACTIVITY_BUY_GOODS_CMD, SERVER_ACTIVITY_BUY_GOODS, body)
			if err != nil {
				log.Printf("[%s] [Qiling] mua gói %s (%d) lỗi: %v", p.Username, row.GetName(), row.GetId(), err)
				break
			}
			var response proto.RspBuyActivityGoods
			if pb.Unmarshal(data, &response) != nil || response.GetRet() != 0 {
				log.Printf("[%s] [Qiling] máy chủ từ chối gói %s (%d)", p.Username, row.GetName(), row.GetId())
				break
			}
			bought[row.GetId()]++
			p.RbtMu.Lock()
			p.qlMallBought[row.GetId()] = bought[row.GetId()]
			p.RbtMu.Unlock()
			kindName := "miễn phí"
			switch kind {
			case qilingarea.MallAdGated:
				kindName = "quảng cáo (theo cooldown máy chủ)"
			case qilingarea.MallJade:
				kindName = "Tiên Ngọc"
			}
			log.Printf("[%s] [Qiling] đã mua gói %s (%s), lượt %d/%d", p.Username, row.GetName(), kindName, bought[row.GetId()], limit)
		}
	}
}

func (p *EventPlayer) qlClaimReadyRecharge(activityID int32) {
	p.RbtMu.Lock()
	configs := append([]*proto.ActivityConditionConfig(nil), p.qlConditionConfig...)
	data := append([]*proto.ActivityConditionData(nil), p.qlConditionData...)
	claimed := make(map[int32]bool, len(p.qlClaimed))
	for id, yes := range p.qlClaimed {
		claimed[id] = yes
	}
	p.RbtMu.Unlock()
	for _, conditionID := range qilingarea.EligibleRechargeConditions(configs, data) {
		if p.OffQiling.Load() {
			return
		}
		if claimed[conditionID] {
			continue
		}
		body, err := qilingarea.MarshalRechargeClaimRequest(activityID, conditionID)
		if err != nil {
			continue
		}
		responseData, err := p.rbtRT(ACTIVITY_COND_REWARD_CMD, SERVER_ACTIVITY_COND_REWARD, body)
		if err != nil {
			log.Printf("[%s] [Qiling] nhận mốc nạp %d lỗi: %v", p.Username, conditionID, err)
			continue
		}
		var response proto.RspGetActivityConditionReward
		if pb.Unmarshal(responseData, &response) != nil || response.GetRet() != 0 {
			log.Printf("[%s] [Qiling] máy chủ chưa cho nhận mốc nạp %d", p.Username, conditionID)
			continue
		}
		p.RbtMu.Lock()
		if p.qlClaimed == nil {
			p.qlClaimed = make(map[int32]bool)
		}
		p.qlClaimed[conditionID] = true
		p.RbtMu.Unlock()
		rewards := strings.TrimSpace(response.GetRewards())
		if rewards == "" {
			continue
		}
		log.Printf("[%s] [Qiling] đã nhận mốc nạp %d", p.Username, conditionID)
		p.sendNotify(fmt.Sprintf("Tứ Thánh Bảo Vực: đã nhận quà mốc nạp %d", conditionID))
	}
}
