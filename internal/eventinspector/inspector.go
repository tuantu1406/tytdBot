// Package eventinspector provides read-only diagnostics for browser game traffic.
// It never sends game commands: the only CDP messages emitted are Network.enable
// and Page.enable, which enable observation in the attached Edge target.
package eventinspector

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

type Frame struct {
	Timestamp time.Time `json:"timestamp"`
	Direction string    `json:"direction"`
	URL       string    `json:"url,omitempty"`
	Payload   string    `json:"payload"`
	Opcode    string    `json:"opcode,omitempty"`
	Decoded   any       `json:"decoded,omitempty"`
	Sensitive bool      `json:"sensitive,omitempty"`
}
type Timeline struct {
	Frames     []Frame    `json:"frames"`
	Activities []Activity `json:"activities"`
}
type Activity struct {
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"`
	FrameIndex int       `json:"frame_index"`
	New        bool      `json:"new"`
}

var sensitiveKey = regexp.MustCompile(`(?i)(token|authorization|cookie|password|passwd|secret|session|credential|api[_-]?key|signature)`)
var sensitiveText = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+|([A-Za-z0-9_]{20,}:[A-Za-z0-9._~-]{10,})`)

func Redact(v any) any {
	switch x := v.(type) {
	case map[string]any:
		o := make(map[string]any, len(x))
		for k, v := range x {
			if sensitiveKey.MatchString(k) {
				o[k] = "[REDACTED]"
			} else {
				o[k] = Redact(v)
			}
		}
		return o
	case []any:
		o := make([]any, len(x))
		for i := range x {
			o[i] = Redact(x[i])
		}
		return o
	case string:
		return sensitiveText.ReplaceAllString(x, `${1}[REDACTED]`)
	default:
		return v
	}
}

func RedactJSON(data []byte) []byte {
	var v any
	if json.Unmarshal(data, &v) != nil {
		return []byte(sensitiveText.ReplaceAllString(string(data), `${1}[REDACTED]`))
	}
	b, _ := json.Marshal(Redact(v))
	return b
}
func Fingerprint(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:8]) }

// DecodeProto is a schema-free protobuf wire decoder. It intentionally reports
// field numbers and wire values only; callers can layer generated descriptors on top.
func DecodeProto(data []byte) (map[string]any, error) {
	out := map[string]any{}
	for len(data) > 0 {
		n, wt, k := protowire.ConsumeTag(data)
		if k < 0 {
			return nil, protowire.ParseError(k)
		}
		data = data[k:]
		key := fmt.Sprintf("%d", n)
		var value any
		var used int
		switch wt {
		case protowire.VarintType:
			v, k := protowire.ConsumeVarint(data)
			if k < 0 {
				return nil, protowire.ParseError(k)
			}
			value, used = v, k
		case protowire.Fixed32Type:
			v, k := protowire.ConsumeFixed32(data)
			if k < 0 {
				return nil, protowire.ParseError(k)
			}
			value, used = v, k
		case protowire.Fixed64Type:
			v, k := protowire.ConsumeFixed64(data)
			if k < 0 {
				return nil, protowire.ParseError(k)
			}
			value, used = v, k
		case protowire.BytesType:
			v, k := protowire.ConsumeBytes(data)
			if k < 0 {
				return nil, protowire.ParseError(k)
			}
			if json.Valid(v) {
				value = json.RawMessage(RedactJSON(v))
			} else {
				value = "base64:" + Fingerprint(v)
			}
			used = k
		default:
			return nil, fmt.Errorf("unsupported wire type %d", wt)
		}
		if old, ok := out[key]; ok {
			if a, ok := old.([]any); ok {
				out[key] = append(a, value)
			} else {
				out[key] = []any{old, value}
			}
		} else {
			out[key] = value
		}
		data = data[used:]
	}
	return out, nil
}

func AddFrame(t *Timeline, f Frame) {
	raw := []byte(f.Payload)
	if decoded, err := base64.StdEncoding.DecodeString(f.Payload); err == nil && len(decoded) >= 18 && decoded[0] == 0x71 && decoded[1] == 0xab {
		raw = decoded
		f.Opcode = hex.EncodeToString(decoded[8:10])
		if d, err := DecodeProto(decoded[18:]); err == nil && len(d) > 0 {
			f.Decoded = Redact(d)
		}
	}
	f.Payload = string(RedactJSON([]byte(f.Payload)))
	if f.Decoded == nil {
		if d, err := DecodeProto(raw); err == nil && len(d) > 0 {
			f.Decoded = Redact(d)
		}
	}
	f.Sensitive = sensitiveText.MatchString(f.Payload)
	t.Frames = append(t.Frames, f)
}
func ActivityTypes(t *Timeline) []string {
	seen := map[string]bool{}
	for _, a := range t.Activities {
		seen[a.Type] = true
	}
	r := make([]string, 0, len(seen))
	for x := range seen {
		r = append(r, x)
	}
	sort.Strings(r)
	return r
}
func JSONDiff(before, after any) []string {
	var out []string
	diffJSON("", before, after, &out)
	sort.Strings(out)
	return out
}
func diffJSON(path string, a, b any, out *[]string) {
	if bytes.Equal(mustJSON(a), mustJSON(b)) {
		return
	}
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if aok && bok {
		keys := map[string]bool{}
		for k := range am {
			keys[k] = true
		}
		for k := range bm {
			keys[k] = true
		}
		for k := range keys {
			p := k
			if path != "" {
				p = path + "." + k
			}
			diffJSON(p, am[k], bm[k], out)
		}
		return
	}
	*out = append(*out, path)
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func NormalizeActivityType(method, url string, decoded any) string {
	if method != "" {
		return method
	}
	if s, ok := decoded.(map[string]any); ok {
		if v, ok := s["activityType"].(string); ok {
			return v
		}
	}
	return url
}
func init() { _ = strings.Builder{} }
