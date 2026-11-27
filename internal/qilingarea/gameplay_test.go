package qilingarea

import (
	"encoding/hex"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestMonopolyGameplayBuildersMatchClientSchema(t *testing.T) {
	roll, err := MarshalRollDiceRequest(1)
	if err != nil || hex.EncodeToString(roll) != "0801" {
		t.Fatalf("roll=%x err=%v", roll, err)
	}
	if got := ProtocolHex(ProtocolMonopolyRollDice); got != "0003406f" {
		t.Fatalf("roll protocol hex=%s", got)
	}
	if got := ProtocolHex(ProtocolMonopolyEventHandle); got != "00034083" {
		t.Fatalf("event protocol hex=%s", got)
	}

	attack, err := MarshalMonsterAttackEvent(1, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := hex.EncodeToString(attack), "080110021804220408011003"; got != want {
		t.Fatalf("attack=%s want=%s", got, want)
	}
	draw, err := MarshalDrawCardEvent(1, 10, 4, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := hex.EncodeToString(draw), "0801100a1804220408011007"; got != want {
		t.Fatalf("draw=%s want=%s", got, want)
	}
}

func TestMonopolyGameplayBuildersRejectUnsafeIndexes(t *testing.T) {
	if _, err := MarshalRollDiceRequest(0); err == nil {
		t.Fatal("accepted missing activity id")
	}
	if _, err := MarshalMonsterAttackEvent(1, 0, 0); err == nil {
		t.Fatal("accepted missing monster event id")
	}
	if _, err := MarshalMonsterAttackEvent(1, 2, -1); err == nil {
		t.Fatal("accepted negative map position")
	}
	if _, err := MarshalDrawCardEvent(1, 10, 4, -1); err == nil {
		t.Fatal("accepted negative card slot")
	}
}

func TestParseMonopolyRollDiceResponseKnownFieldsAndUnknowns(t *testing.T) {
	monster := appendI32(nil, 2, 41)
	monster = appendBytes(monster, 4, []byte("73"))
	monster = appendI64(monster, 1, 9001)
	explore := appendBytes(nil, 1, []byte("item=1"))
	explore = appendBytes(explore, 2, monster)
	explore = appendI32(explore, 9, 13)
	move := appendBytes(nil, 1, explore)
	move = appendI32(move, 2, 16)
	move = appendI32(move, 4, 2)
	move = appendI32(move, 5, 7)
	move = appendI32(move, 6, 1)
	response := appendI32(nil, 1, 0)
	response = appendI32(response, 2, 21)
	response = appendBytes(response, 3, move)
	response = appendI32(response, 30, 123) // forward-compatible unknown field

	got, err := ParseRollDiceResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ret != 0 || !got.HasMapIndex || got.MapIndex != 21 || !got.HasStrength || got.Strength != 16 || !got.HasEnergyCost || got.TotalCostEnergy != 2 || !got.HasScore || got.Score != 7 || !got.HasNeedMove || !got.NeedMove {
		t.Fatalf("incomplete roll result: %+v", got)
	}
	if got.EventID != 13 || got.Reward != "item=1" || got.MonsterID != 41 || got.MonsterUID != 9001 || got.MonsterLeftHP != "73" {
		t.Fatalf("incomplete landing result: %+v", got)
	}
	if _, err := ParseRollDiceResponse(appendI32(nil, 2, 1)); err == nil {
		t.Fatal("accepted response missing required ret")
	}
}

func appendI32(dst []byte, field protowire.Number, value int32) []byte {
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	return protowire.AppendVarint(dst, uint64(uint32(value)))
}

func appendI64(dst []byte, field protowire.Number, value int64) []byte {
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	return protowire.AppendVarint(dst, uint64(value))
}

func appendBytes(dst []byte, field protowire.Number, value []byte) []byte {
	dst = protowire.AppendTag(dst, field, protowire.BytesType)
	return protowire.AppendBytes(dst, value)
}
