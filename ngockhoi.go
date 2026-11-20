package tytd_go

import "google.golang.org/protobuf/encoding/protowire"

// Ngự Khôi Tranh Phong (activity type 321) — verified wire facts.
const (
	ngocKhoiActivityType     int32 = 321
	ngocKhoiGuildConditionID int32 = 10003
	ngocKhoiEnterCmd               = "00005141"
)

// ngocKhoiEnterPayload builds the request captured while opening the event.
// Field 1 is activity type, field 2 is the client mode, field 3 is the server
// activity id. mode meanings are intentionally not inferred here; callers
// must choose the mode observed from the UI action they are reproducing.
func ngocKhoiEnterPayload(mode, activityID int32) []byte {
	b := make([]byte, 0, 12)
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(ngocKhoiActivityType))
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(mode))
	b = protowire.AppendTag(b, 3, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(activityID))
	return b
}
