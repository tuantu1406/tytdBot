package eventinspector

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"tytd_go/internal/proto"

	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

const (
	OpcodeActivityList       = "03e9"     // incoming 21001
	OpcodeActivityListUpdate = "040c"     // incoming 21036
	OpcodeActivityDetail     = "03eb"     // incoming 21003 response
	OpcodeActivityDetailReq  = "0000520b" // outgoing 21003 request
)

type PacketRecord struct {
	Timestamp  time.Time  `json:"timestamp"`
	Direction  string     `json:"direction"`
	Opcode     string     `json:"opcode"`
	Known      bool       `json:"known"`
	Length     int        `json:"length"`
	SHA256     string     `json:"sha256"`
	PairID     string     `json:"pair_id,omitempty"`
	Message    string     `json:"message,omitempty"`
	Activities []Activity `json:"activities,omitempty"`
	Diff       []string   `json:"diff,omitempty"`
}

type AnalyzerReport struct {
	Packets          []PacketRecord `json:"packets"`
	NewActivityTypes []int32        `json:"new_activity_types,omitempty"`
}

type Analyzer struct {
	mu         sync.Mutex
	report     AnalyzerReport
	seq        uint64
	pending    map[int32]string
	knownTypes map[int32]bool
	state      map[int32]any
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{pending: map[int32]string{}, knownTypes: map[int32]bool{}, state: map[int32]any{}}
}

func (a *Analyzer) Observe(direction, opcode string, payload []byte) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.seq++
	r := PacketRecord{Timestamp: time.Now().UTC(), Direction: direction, Opcode: opcode, Length: len(payload), SHA256: Fingerprint(payload)}
	switch opcode {
	case OpcodeActivityList, OpcodeActivityListUpdate:
		var msg proto.PushActivityList
		if err := pb.Unmarshal(payload, &msg); err == nil {
			r.Known = true
			r.Message = "PushActivityList"
			for _, c := range msg.MainConfig {
				if c.GetActivityId() == 0 {
					continue
				}
				r.Activities = append(r.Activities, Activity{Type: fmt.Sprint(c.GetType()), FrameIndex: len(a.report.Packets)})
				a.recordStateLocked(c.GetActivityId(), c, &r)
			}
		}
	case OpcodeActivityDetailReq:
		var msg proto.ReqGetActivityDetail
		if err := pb.Unmarshal(payload, &msg); err == nil {
			r.Known = true
			r.Message = "ReqGetActivityDetail"
			a.pending[msg.GetActivityId()] = fmt.Sprintf("p-%d", a.seq)
			r.PairID = a.pending[msg.GetActivityId()]
		}
	case OpcodeActivityDetail:
		var msg proto.RspGetActivityDetail
		if err := pb.Unmarshal(payload, &msg); err == nil {
			r.Known = true
			r.Message = "RspGetActivityDetail"
			c := msg.GetActivity().GetDetailConfig().GetCommonConfig().GetMainConfig()
			if c != nil {
				id := c.GetActivityId()
				r.PairID = a.pending[id]
				delete(a.pending, id)
				r.Activities = append(r.Activities, Activity{Type: fmt.Sprint(c.GetType()), FrameIndex: len(a.report.Packets)})
				a.recordStateLocked(id, c, &r)
			}
		}
	}
	a.report.Packets = append(a.report.Packets, r)
}

func (a *Analyzer) recordStateLocked(id int32, msg pb.Message, r *PacketRecord) {
	b, err := protojson.Marshal(msg)
	if err != nil {
		return
	}
	var next any
	if json.Unmarshal(RedactJSON(b), &next) != nil {
		return
	}
	if old, ok := a.state[id]; ok {
		r.Diff = JSONDiff(old, next)
	}
	a.state[id] = next
	if c, ok := msg.(*proto.ActivityMainConfig); ok && !a.knownTypes[c.GetType()] {
		a.knownTypes[c.GetType()] = true
		r.Activities[len(r.Activities)-1].New = true
		a.report.NewActivityTypes = append(a.report.NewActivityTypes, c.GetType())
	}
}

func (a *Analyzer) Snapshot() AnalyzerReport {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.report
	out.Packets = append([]PacketRecord(nil), a.report.Packets...)
	out.NewActivityTypes = append([]int32(nil), a.report.NewActivityTypes...)
	return out
}
func (a *Analyzer) WriteJSON(w io.Writer) error { return json.NewEncoder(w).Encode(a.Snapshot()) }
func (a *Analyzer) WriteJSONL(w io.Writer) error {
	for _, p := range a.Snapshot().Packets {
		b, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if _, err = fmt.Fprintln(w, string(b)); err != nil {
			return err
		}
	}
	return nil
}

var moduleIdent = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func GenerateModuleSkeleton(name string, verified bool) (string, error) {
	if !verified {
		return "", fmt.Errorf("refusing to generate module: packet data is not marked verified")
	}
	if !moduleIdent.MatchString(name) {
		return "", fmt.Errorf("invalid module name %q", name)
	}
	return fmt.Sprintf("// Generated read-only scaffold for verified event %s.\n// Add request encoders only after an offline fixture is pinned.\npackage main\n\nconst eventModuleName = %q\n", name, name), nil
}

func HexOpcode(opcode string) string { b, _ := hex.DecodeString(opcode); return hex.EncodeToString(b) }
