package eventinspector

import (
	"bytes"
	"testing"

	pb "google.golang.org/protobuf/proto"
	"tytd_go/internal/proto"
)

func TestAnalyzerLinksActivityDetailAndDiffsState(t *testing.T) {
	a := NewAnalyzer()
	request, err := pb.Marshal(&proto.ReqGetActivityDetail{ActivityId: 42, Version: pb.String("0")})
	if err != nil {
		t.Fatal(err)
	}
	a.Observe("sent", OpcodeActivityDetailReq, request)
	first := &proto.ActivityMainConfig{ActivityId: pb.Int32(42), Type: pb.Int32(901), BeginTime: pb.Int64(10)}
	second := &proto.ActivityMainConfig{ActivityId: pb.Int32(42), Type: pb.Int32(901), BeginTime: pb.Int64(20)}
	for _, c := range []*proto.ActivityMainConfig{first, second} {
		body, err := pb.Marshal(&proto.RspGetActivityDetail{Activity: &proto.ActivityCommonData{ActivityId: pb.Int32(42), DetailConfig: &proto.ActivityDetailConfig{CommonConfig: &proto.ActivityCommonConfig{MainConfig: c}}}})
		if err != nil {
			t.Fatal(err)
		}
		a.Observe("received", OpcodeActivityDetail, body)
	}
	r := a.Snapshot()
	if len(r.Packets) != 3 || r.Packets[1].PairID == "" || len(r.Packets[2].Diff) == 0 {
		t.Fatalf("unexpected report: %+v", r)
	}
	if len(r.NewActivityTypes) != 1 || r.NewActivityTypes[0] != 901 {
		t.Fatalf("new types: %+v", r.NewActivityTypes)
	}
}

func TestAnalyzerWritesJSONLWithoutRawPayload(t *testing.T) {
	a := NewAnalyzer()
	a.Observe("received", "ffff", []byte("secret-token"))
	var b bytes.Buffer
	if err := a.WriteJSONL(&b); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b.Bytes(), []byte("secret-token")) {
		t.Fatal("raw payload leaked")
	}
}

func TestGenerateModuleSkeletonRequiresVerification(t *testing.T) {
	if _, err := GenerateModuleSkeleton("new_event", false); err == nil {
		t.Fatal("unverified module was generated")
	}
	if got, err := GenerateModuleSkeleton("new_event", true); err != nil || !bytes.Contains([]byte(got), []byte("new_event")) {
		t.Fatalf("got %q err=%v", got, err)
	}
}
