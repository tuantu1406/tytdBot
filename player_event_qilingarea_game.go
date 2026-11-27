package tytd_go

import (
	"errors"
	"fmt"
	"time"
	"tytd_go/internal/qilingarea"
)

var errQilingNoActiveActivity = errors.New("không có sự kiện Tứ Thánh Bảo Vực đang mở trên phiên này")

func (p *EventPlayer) qlActiveActivityID() int32 {
	var active activityEntry
	now := time.Now()
	for _, entry := range p.ActList.byType(qilingAreaType) {
		if entry.status(now) == "active" && entry.ActivityID > active.ActivityID {
			active = entry
		}
	}
	return active.ActivityID
}

// QilingRollDiceOnce sends exactly one 213103 roll for the selected, active
// Type 131 event. It never retries, replenishes energy, or starts autoplay.
func (p *EventPlayer) QilingRollDiceOnce() (qilingarea.RollDiceResult, error) {
	if p == nil || !p.Running || p.ctx == nil {
		return qilingarea.RollDiceResult{}, errors.New("phiên game chưa kết nối")
	}
	if !p.qlGameBusy.CompareAndSwap(false, true) {
		return qilingarea.RollDiceResult{}, errors.New("đang chờ thao tác Tứ Thánh Bảo Vực trước đó")
	}
	defer p.qlGameBusy.Store(false)
	activityID := p.qlActiveActivityID()
	if activityID <= 0 {
		return qilingarea.RollDiceResult{}, errQilingNoActiveActivity
	}
	body, err := qilingarea.MarshalRollDiceRequest(activityID)
	if err != nil {
		return qilingarea.RollDiceResult{}, err
	}
	data, err := p.rbtRT(MONOPOLY_ROLL_DICE_CMD, SERVER_MONOPOLY_ROLL_DICE, body)
	if err != nil {
		return qilingarea.RollDiceResult{}, err
	}
	result, err := qilingarea.ParseRollDiceResponse(data)
	if err != nil {
		return qilingarea.RollDiceResult{}, err
	}
	if result.Ret != 0 {
		return result, fmt.Errorf("máy chủ từ chối gieo xúc xắc (ret=%d)", result.Ret)
	}
	return result, nil
}
