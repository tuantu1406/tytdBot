package tytd_go

import (
	"log"
	"time"

	"tytd_go/internal/pack"
	"tytd_go/internal/proto"

	pb "google.golang.org/protobuf/proto"
)

const ngocKhoiActivityType int32 = 321

// ngocKhoiHandleDetail consumes only the server-authored 21003 config for
// activity type 321. It never treats UI labels as authority.
func (p *EventPlayer) ngocKhoiHandleDetail(data []byte) {
	var rsp proto.RspGetActivityDetail
	if err := pb.Unmarshal(data, &rsp); err != nil || rsp.GetRet() != 0 {
		return
	}
	activity := rsp.GetActivity()
	main := activity.GetDetailConfig().GetCommonConfig().GetMainConfig()
	if main == nil || main.GetType() != ngocKhoiActivityType {
		return
	}
	rows := make([]*proto.ActivityMallTempMsg, 0)
	for _, cfg := range activity.GetDetailConfig().GetCommonConfig().GetMallConfig() {
		if row := cfg.GetMallTempMsg(); row != nil {
			rows = append(rows, row)
		}
	}
	bought := make(map[int32]int64)
	for _, item := range activity.GetMallBuyCountList() {
		bought[item.GetMallId()] = item.GetCount()
	}
	p.NgocKhoiMu.Lock()
	p.ngocKhoiID = main.GetActivityId()
	if len(rows) > 0 {
		p.ngocKhoiRows = rows
		p.ngocKhoiBought = bought
		p.ngocKhoiDetail = time.Now()
	}
	p.NgocKhoiMu.Unlock()
	if len(rows) > 0 {
		log.Printf("[%s] [NgocKhoi] đã đọc %d gói, đã mua hôm nay: %v", p.Username, len(rows), bought)
	}
}

// ngocKhoiShopLoop sends at most one verified purchase every 5 seconds. The
// response handler below advances the FIFO only on ret=0, so a refused ad or
// insufficient Tiên Ngọc is not counted locally and can be retried later.
func (p *EventPlayer) ngocKhoiShopLoop() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-t.C:
			p.ngocKhoiShopStep()
		}
	}
}

func (p *EventPlayer) ngocKhoiShopStep() {
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
		body, err := pb.Marshal(&proto.ReqGetActivityDetail{ActivityId: id})
		if err == nil {
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
	p.write(pack.Pack(ACTIVITY_BUY_GOODS_CMD, mustMarshalNgocKhoi(req), p.UID))
}

func mustMarshalNgocKhoi(req *proto.ReqBuyActivityGoods) []byte {
	b, _ := marshalNgocKhoiShopRequest(req)
	return b
}

// ngocKhoiHandleBuyResponse claims the shared 03ed response only when this
// engine has an outstanding request. Returning false lets existing event
// engines handle their own 21005 seat.
func (p *EventPlayer) ngocKhoiHandleBuyResponse(data []byte) bool {
	p.NgocKhoiMu.Lock()
	if len(p.ngocKhoiPending) == 0 {
		p.NgocKhoiMu.Unlock()
		return false
	}
	mallID := p.ngocKhoiPending[0]
	p.ngocKhoiPending = p.ngocKhoiPending[1:]
	p.NgocKhoiMu.Unlock()
	var rsp proto.RspBuyActivityGoods
	if err := pb.Unmarshal(data, &rsp); err != nil {
		return true
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
	return true
}
