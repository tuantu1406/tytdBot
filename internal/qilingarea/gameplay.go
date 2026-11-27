package qilingarea

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// The client runtime declares these protocol IDs in decimal. The websocket
// frame carries the full uint32 (unlike legacy 16-bit game commands).
const (
	ProtocolMonopolyRollDice    uint32 = 213103
	ProtocolMonopolyEventHandle uint32 = 213123
	MonopolyActionAttackMonster int32  = 3
)

func ProtocolHex(protocol uint32) string { return fmt.Sprintf("%08x", protocol) }

// MarshalRollDiceRequest follows MonopolyRollDiceReq: activityId=1.
func MarshalRollDiceRequest(activityID int32) ([]byte, error) {
	if activityID <= 0 {
		return nil, errors.New("activityId must be positive")
	}
	return appendInt32Field(nil, 1, activityID), nil
}

// MarshalMonsterAttackEvent builds the nested MonopolyEventHandleReq carrying
// MonopolyEventActionReqMsg{activityId, action=3}. It performs no I/O.
func MarshalMonsterAttackEvent(activityID, eventID, mapIndex int32) ([]byte, error) {
	if activityID <= 0 || eventID <= 0 || mapIndex < 0 {
		return nil, errors.New("activityId/eventId must be positive and mapIndex non-negative")
	}
	inner := appendInt32Field(nil, 1, activityID)
	inner = appendInt32Field(inner, 2, MonopolyActionAttackMonster)
	return marshalEventHandle(activityID, eventID, mapIndex, inner), nil
}

// MarshalDrawCardEvent builds MonopolyDrawReq{activityId, index} inside the
// MonopolyEventHandleReq envelope. cardIndex is the client-selected card slot.
func MarshalDrawCardEvent(activityID, eventID, mapIndex, cardIndex int32) ([]byte, error) {
	if activityID <= 0 || eventID <= 0 || mapIndex < 0 || cardIndex < 0 {
		return nil, errors.New("activityId/eventId must be positive and indexes non-negative")
	}
	inner := appendInt32Field(nil, 1, activityID)
	inner = appendInt32Field(inner, 2, cardIndex)
	return marshalEventHandle(activityID, eventID, mapIndex, inner), nil
}

func marshalEventHandle(activityID, eventID, mapIndex int32, inner []byte) []byte {
	var out []byte
	out = appendInt32Field(out, 1, activityID)
	out = appendInt32Field(out, 2, eventID)
	out = appendInt32Field(out, 3, mapIndex)
	out = protowire.AppendTag(out, 4, protowire.BytesType)
	return protowire.AppendBytes(out, inner)
}

func appendInt32Field(dst []byte, field protowire.Number, value int32) []byte {
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	return protowire.AppendVarint(dst, uint64(uint32(value)))
}

type RollDiceResult struct {
	Ret             int32
	MapIndex        int32
	HasMapIndex     bool
	Strength        int32
	HasStrength     bool
	TotalCostEnergy int32
	HasEnergyCost   bool
	Score           int32
	HasScore        bool
	NeedMove        bool
	HasNeedMove     bool
	EventID         int32
	Reward          string
	MonsterID       int32
	MonsterUID      int64
	MonsterLeftHP   string
}

// ParseRollDiceResponse reads only documented fields and skips unknown ones.
func ParseRollDiceResponse(data []byte) (RollDiceResult, error) {
	var result RollDiceResult
	seenRet := false
	err := consumeMessage(data, func(n protowire.Number, typ protowire.Type, raw []byte, value uint64) error {
		switch n {
		case 1:
			if typ != protowire.VarintType {
				return errors.New("roll response ret has wrong wire type")
			}
			result.Ret, seenRet = int32(value), true
		case 2:
			if typ != protowire.VarintType {
				return errors.New("roll response mapIndex has wrong wire type")
			}
			result.MapIndex, result.HasMapIndex = int32(value), true
		case 3:
			if typ != protowire.BytesType {
				return errors.New("roll response moveResult has wrong wire type")
			}
			return parseMoveResult(raw, &result)
		}
		return nil
	})
	if err != nil {
		return RollDiceResult{}, err
	}
	if !seenRet {
		return RollDiceResult{}, errors.New("roll response missing ret")
	}
	return result, nil
}

func parseMoveResult(data []byte, result *RollDiceResult) error {
	return consumeMessage(data, func(n protowire.Number, typ protowire.Type, raw []byte, value uint64) error {
		switch n {
		case 1:
			if typ != protowire.BytesType {
				return errors.New("move data has wrong wire type")
			}
			return parseExploreResult(raw, result)
		case 2:
			if typ != protowire.VarintType {
				return errors.New("strength has wrong wire type")
			}
			result.Strength, result.HasStrength = int32(value), true
		case 4:
			if typ != protowire.VarintType {
				return errors.New("energy cost has wrong wire type")
			}
			result.TotalCostEnergy, result.HasEnergyCost = int32(value), true
		case 5:
			if typ != protowire.VarintType {
				return errors.New("score has wrong wire type")
			}
			result.Score, result.HasScore = int32(value), true
		case 6:
			if typ != protowire.VarintType {
				return errors.New("needMove has wrong wire type")
			}
			result.NeedMove, result.HasNeedMove = value != 0, true
		}
		return nil
	})
}

func parseExploreResult(data []byte, result *RollDiceResult) error {
	return consumeMessage(data, func(n protowire.Number, typ protowire.Type, raw []byte, value uint64) error {
		switch n {
		case 1:
			if typ != protowire.BytesType {
				return errors.New("reward has wrong wire type")
			}
			result.Reward = string(raw)
		case 2:
			if typ != protowire.BytesType {
				return errors.New("monsterInfo has wrong wire type")
			}
			return parseMonsterInfo(raw, result)
		case 9:
			if typ != protowire.VarintType {
				return errors.New("eventId has wrong wire type")
			}
			result.EventID = int32(value)
		}
		return nil
	})
}

func parseMonsterInfo(data []byte, result *RollDiceResult) error {
	return consumeMessage(data, func(n protowire.Number, typ protowire.Type, raw []byte, value uint64) error {
		switch n {
		case 1:
			if typ != protowire.VarintType {
				return errors.New("monsterUid has wrong wire type")
			}
			result.MonsterUID = int64(value)
		case 2:
			if typ != protowire.VarintType {
				return errors.New("monsterId has wrong wire type")
			}
			result.MonsterID = int32(value)
		case 4:
			if typ != protowire.BytesType {
				return errors.New("monster leftHp has wrong wire type")
			}
			result.MonsterLeftHP = string(raw)
		}
		return nil
	})
}

func consumeMessage(data []byte, visit func(protowire.Number, protowire.Type, []byte, uint64) error) error {
	for len(data) > 0 {
		n, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return protowire.ParseError(tagLen)
		}
		data = data[tagLen:]
		var raw []byte
		var value uint64
		var consumed int
		switch typ {
		case protowire.VarintType:
			value, consumed = protowire.ConsumeVarint(data)
		case protowire.BytesType:
			raw, consumed = protowire.ConsumeBytes(data)
		default:
			consumed = protowire.ConsumeFieldValue(n, typ, data)
		}
		if consumed < 0 {
			return protowire.ParseError(consumed)
		}
		if typ == protowire.VarintType || typ == protowire.BytesType {
			if err := visit(n, typ, raw, value); err != nil {
				return err
			}
		}
		data = data[consumed:]
	}
	return nil
}
