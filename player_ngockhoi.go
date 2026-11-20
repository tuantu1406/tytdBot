package tytd_go

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	pb "google.golang.org/protobuf/proto"
	"tytd_go/internal/pack"
	"tytd_go/internal/proto"
)

func (p *Player) ngocKhoiPlayerHandleDetail(data []byte) {
	var rsp proto.RspGetActivityDetail
	if pb.Unmarshal(data, &rsp) != nil || rsp.GetRet() != 0 {
		return
	}
	a := rsp.GetActivity()
	main := a.GetDetailConfig().GetCommonConfig().GetMainConfig()
	if main == nil || main.GetType() != ngocKhoiActivityType {
		return
	}
	rows := make([]*proto.ActivityMallTempMsg, 0)
	for _, cfg := range a.GetDetailConfig().GetCommonConfig().GetMallConfig() {
		if row := cfg.GetMallTempMsg(); row != nil {
			rows = append(rows, row)
		}
	}
	bought := map[int32]int64{}
	for _, item := range a.GetMallBuyCountList() {
		bought[item.GetMallId()] = item.GetCount()
	}
	p.NgocKhoiMu.Lock()
	p.ngocKhoiID = main.GetActivityId()
	if len(rows) > 0 {
		p.ngocKhoiRows, p.ngocKhoiBought, p.ngocKhoiDetail = rows, bought, time.Now()
	}
	p.NgocKhoiMu.Unlock()
	if len(rows) > 0 {
		log.Printf("[%s] [NgocKhoi] đã đọc %d gói, đã mua hôm nay: %v", p.Username, len(rows), bought)
	}
	p.ngocKhoiPlayerClaimGuild(main.GetActivityId())
}

func (p *Player) ngocKhoiPlayerClaimGuild(activityID int32) {
	today := time.Now().In(ictZone).Format("2006-01-02")
	p.NgocKhoiMu.Lock()
	if p.ngocKhoiGuildDay == today {
		p.NgocKhoiMu.Unlock()
		return
	}
	p.NgocKhoiMu.Unlock()
	claimed := false
	for _, raw := range strings.Split(ngocKhoiGuildConditionIDs, ",") {
		cid, _ := strconv.Atoi(raw)
		rewards, err := p.swClaimCond(activityID, int32(cid))
		if err != nil && errors.Is(err, errStTimeout) {
			time.Sleep(500 * time.Millisecond)
			rewards, err = p.swClaimCond(activityID, int32(cid))
		}
		if err != nil {
			log.Printf("[%s] [NgocKhoi] mốc %d lỗi/từ chối: %v", p.Username, cid, err)
			continue
		}
		if rewards == "" {
			continue
		}
		claimed = true
		log.Printf("[%s] [NgocKhoi] đã nhận mốc bang conditionId=%d rewards=%s", p.Username, cid, rewards)
	}
	if !claimed {
		return
	}
	p.NgocKhoiMu.Lock()
	p.ngocKhoiGuildDay = today
	p.NgocKhoiMu.Unlock()
}

func (p *Player) ngocKhoiPlayerLoop() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-t.C:
			p.NgocKhoiMu.Lock()
			id := p.ngocKhoiID
			p.NgocKhoiMu.Unlock()
			if id != 0 {
				p.ngocKhoiVerifiedCycle(id, "continuous")
			}
			p.ngocKhoiPlayerStep()
		}
	}
}

func (p *Player) ngocKhoiPlayerStep() {
	entries := p.ActList.byType(ngocKhoiActivityType)
	if len(entries) == 0 {
		return
	}
	p.NgocKhoiMu.Lock()
	if len(p.ngocKhoiPending) > 0 {
		p.NgocKhoiMu.Unlock()
		return
	}
	id, rows, bought, detail := p.ngocKhoiID, p.ngocKhoiRows, p.ngocKhoiBought, p.ngocKhoiDetail
	p.NgocKhoiMu.Unlock()
	if id == 0 {
		id = entries[0].ActivityID
	}
	if len(rows) == 0 || time.Since(detail) > 5*time.Minute {
		if body, err := pb.Marshal(&proto.ReqGetActivityDetail{ActivityId: id}); err == nil {
			p.write(pack.Pack(ACTIVITY_GET_DATA_CMD, body, p.UID))
		}
		return
	}
	requests := ngocKhoiShopRequests(id, rows, bought)
	if len(requests) == 0 {
		return
	}
	req := requests[0]
	p.NgocKhoiMu.Lock()
	p.ngocKhoiPending = append(p.ngocKhoiPending, req.GetMallId())
	p.NgocKhoiMu.Unlock()
	if body, err := marshalNgocKhoiShopRequest(req); err == nil {
		p.write(pack.Pack(ACTIVITY_BUY_GOODS_CMD, body, p.UID))
	}
}

func (p *Player) ngocKhoiPlayerHandleBuy(data []byte) {
	p.NgocKhoiMu.Lock()
	if len(p.ngocKhoiPending) == 0 {
		p.NgocKhoiMu.Unlock()
		return
	}
	mallID := p.ngocKhoiPending[0]
	p.ngocKhoiPending = p.ngocKhoiPending[1:]
	p.NgocKhoiMu.Unlock()
	var rsp proto.RspBuyActivityGoods
	if pb.Unmarshal(data, &rsp) != nil {
		return
	}
	if rsp.GetRet() == 0 {
		p.NgocKhoiMu.Lock()
		if p.ngocKhoiBought == nil {
			p.ngocKhoiBought = map[int32]int64{}
		}
		p.ngocKhoiBought[mallID]++
		p.NgocKhoiMu.Unlock()
		log.Printf("[%s] [NgocKhoi] mua thành công mallId=%d", p.Username, mallID)
	} else {
		log.Printf("[%s] [NgocKhoi] mallId=%d bị từ chối ret=%d", p.Username, mallID, rsp.GetRet())
	}
}
