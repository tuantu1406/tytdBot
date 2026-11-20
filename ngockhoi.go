package tytd_go

import "google.golang.org/protobuf/encoding/protowire"

// Ngự Khôi Tranh Phong (activity type 321) — verified wire facts.
const (
	ngocKhoiActivityType        int32 = 321
	ngocKhoiGuildConditionID    int32 = 10003
	ngocKhoiEnterCmd                  = "00005141"
	ngocKhoiTreeDonateCmd             = "000397ff" // wire 8ac3, captured 2026-11-20
	ngocKhoiTreeDonateResp            = "7d83"
	ngocKhoiTreeClaimCmd              = "000397fe" // wire 8ac2, captured 2026-11-20
	ngocKhoiTreeClaimResp             = "7d82"
	ngocKhoiTreeLevelRewardCmd        = "000397fd" // wire 8ac1, captured 2026-11-20
	ngocKhoiTreeLevelRewardResp       = "7d81"
	ngocKhoiEquipReorderCmd           = "000397f4" // wire 8ab4, captured 2026-11-20
	ngocKhoiEquipReorderResp          = "7d74"
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

// ngocKhoiTreeDonatePayload builds the captured free/authorised tree action:
// {activityId, itemId, count}. The item id is deliberately supplied by the
// caller so the item-ban boundary cannot be bypassed by a guessed default.
func ngocKhoiTreeDonatePayload(activityID, itemID, count int32) []byte {
	b := make([]byte, 0, 16)
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(activityID))
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(itemID))
	b = protowire.AppendTag(b, 3, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(count))
	return b
}

// ngocKhoiTreeClaimPayload builds the captured claim request
// {activityId, claimMode}. The observed claimMode is 0 and remains explicit.
func ngocKhoiTreeClaimPayload(activityID, claimMode int32) []byte {
	b := make([]byte, 0, 12)
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(activityID))
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(claimMode))
	return b
}

// ngocKhoiTreeLevelRewardPayload builds the level-up reward claim
// {activityId} captured from the event panel.
func ngocKhoiTreeLevelRewardPayload(activityID int32) []byte {
	b := make([]byte, 0, 8)
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(activityID))
	return b
}

// ngocKhoiEquipReorderPayload builds the captured reorder request.
func ngocKhoiEquipReorderPayload(activityID, mode, from, to, slot int32) []byte {
	b := make([]byte, 0, 16)
	for field, value := range []int32{activityID, mode, from, to, slot} {
		b = protowire.AppendTag(b, protowire.Number(field+1), protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(value))
	}
	return b
}
