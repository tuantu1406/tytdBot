package eventinspector

import (
	"encoding/json"
	"testing"
)

func TestRedactJSON(t *testing.T) {
	got := string(RedactJSON([]byte(`{"token":"abc","nested":{"password":"x"},"name":"ok"}`)))
	var v map[string]any
	if json.Unmarshal([]byte(got), &v) != nil || v["token"] != "[REDACTED]" || v["name"] != "ok" {
		t.Fatalf("redaction failed: %s", got)
	}
}
func TestDecodeProto(t *testing.T) {
	got, err := DecodeProto([]byte{0x08, 0x96, 0x01, 0x12, 0x02, 'o', 'k'})
	if err != nil || got["1"] != uint64(150) || got["2"] != "base64:2689367b205c16ce" {
		t.Fatalf("decode=%v err=%v", got, err)
	}
}
func TestJSONDiff(t *testing.T) {
	d := JSONDiff(map[string]any{"a": 1, "b": 2}, map[string]any{"a": 1, "b": 3, "c": 4})
	if len(d) != 2 || d[0] != "b" || d[1] != "c" {
		t.Fatal(d)
	}
}
