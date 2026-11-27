package tytd_go

import (
	"context"
	"testing"
	"time"

	"tytd_go/internal/pack"
	gamepb "tytd_go/internal/proto"
	"tytd_go/internal/qilingarea"

	pb "google.golang.org/protobuf/proto"
)

func TestQilingRollDiceOnceUsesActiveActivityAndFullProtocolID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Now().UnixMilli()
	activityID := int32(9064648)
	typ := qilingAreaType
	begin, end := now-60_000, now+60_000
	list, err := pb.Marshal(&gamepb.PushActivityList{MainConfig: []*gamepb.ActivityMainConfig{{
		ActivityId: pb.Int32(activityID), Type: pb.Int32(typ), BeginTime: pb.Int64(begin), EndTime: pb.Int64(end),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	ep := &EventPlayer{Running: true, ctx: ctx, writeChan: make(chan []byte, 1)}
	ep.ActList.applyList("test", list, true)
	type callResult struct {
		result qilingarea.RollDiceResult
		err    error
	}
	done := make(chan callResult, 1)
	go func() { result, callErr := ep.QilingRollDiceOnce(); done <- callResult{result, callErr} }()

	frame := <-ep.writeChan
	ok, command, request := pack.Unpack(frame)
	if !ok || command != SERVER_MONOPOLY_ROLL_DICE {
		t.Fatalf("request command=%s ok=%v", command, ok)
	}
	want, _ := qilingarea.MarshalRollDiceRequest(activityID)
	if string(request) != string(want) {
		t.Fatalf("request=%x want=%x", request, want)
	}
	response := qlTestRollResponse(19, 13, 2, 6)
	ep.rbtHandleFrameEP(SERVER_MONOPOLY_ROLL_DICE, response)
	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.result.Ret != 0 || got.result.MapIndex != 19 || got.result.Strength != 13 || got.result.TotalCostEnergy != 2 || got.result.Score != 6 {
		t.Fatalf("unexpected result: %+v", got.result)
	}
}

func TestQilingRollDiceOnceRequiresLiveEventAndSerializesActions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ep := &EventPlayer{Running: true, ctx: ctx, writeChan: make(chan []byte, 1)}
	if _, err := ep.QilingRollDiceOnce(); err == nil {
		t.Fatal("roll accepted without an active Type 131 event")
	}
	ep.qlGameBusy.Store(true)
	if _, err := ep.QilingRollDiceOnce(); err == nil {
		t.Fatal("overlapping roll accepted")
	}
	if len(ep.writeChan) != 0 {
		t.Fatal("request was sent without a valid action seat")
	}
}

func qlTestRollResponse(mapIndex, strength, energyCost, score int32) []byte {
	move := qilingAppendI32(nil, 2, strength)
	move = qilingAppendI32(move, 4, energyCost)
	move = qilingAppendI32(move, 5, score)
	response := qilingAppendI32(nil, 1, 0)
	response = qilingAppendI32(response, 2, mapIndex)
	return qilingAppendBytes(response, 3, move)
}

func qilingAppendI32(dst []byte, field int32, value int32) []byte {
	// The tiny fixture uses the same protobuf varint wire encoding as the
	// offline qilingarea tests; builders themselves are covered there.
	dst = append(dst, byte(field<<3))
	for value >= 0x80 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}

func qilingAppendBytes(dst []byte, field int32, value []byte) []byte {
	dst = append(dst, byte(field<<3|2), byte(len(value)))
	return append(dst, value...)
}
