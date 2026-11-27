package pack

import "testing"

func TestPackUnpackPreserves32BitProtocolAndLegacy16BitCommands(t *testing.T) {
	for _, tc := range []struct{ command, want string }{
		{command: "0000520b", want: "520b"},     // legacy activity detail 21003
		{command: "0003406f", want: "0003406f"}, // Monopoly roll dice 213103
		{command: "00034083", want: "00034083"}, // Monopoly event handle 213123
	} {
		wire := Pack(tc.command, []byte{1, 2}, 123)
		ok, got, payload := Unpack(wire)
		if !ok || got != tc.want {
			t.Fatalf("Unpack(%s)=(%v,%s), want %s", tc.command, ok, got, tc.want)
		}
		if len(payload) != 2 || payload[0] != 1 || payload[1] != 2 {
			t.Fatalf("payload=%x", payload)
		}
	}
}
