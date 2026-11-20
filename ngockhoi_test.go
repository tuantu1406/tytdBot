package tytd_go

import "testing"

func TestNgocKhoiEnterPayload(t *testing.T) {
	got := ngocKhoiEnterPayload(1, 9064454)
	want := []byte{0x08, 0xc1, 0x02, 0x10, 0x01, 0x18, 0x86, 0xa0, 0xa9, 0x04}
	if string(got) != string(want) {
		t.Fatalf("payload=%x want=%x", got, want)
	}
}

func TestNgocKhoiTreeDonatePayload(t *testing.T) {
	b := ngocKhoiTreeDonatePayload(9064454, 105324, 1)
	if len(b) == 0 || b[0] != 0x08 || b[len(b)-1] != 0x01 {
		t.Fatalf("unexpected tree donate payload: %x", b)
	}
	if got := ngocKhoiTreeDonateCmd; got != "000397ff" {
		t.Fatalf("command=%s", got)
	}
}

func TestNgocKhoiTreeClaimPayload(t *testing.T) {
	b := ngocKhoiTreeClaimPayload(9064454, 0)
	if len(b) != 7 || b[0] != 0x08 || b[5] != 0x10 || b[6] != 0x00 {
		t.Fatalf("unexpected tree claim payload: %x", b)
	}
	if ngocKhoiTreeClaimCmd != "000397fe" || ngocKhoiTreeClaimResp != "7d82" {
		t.Fatalf("unexpected tree claim command/response")
	}
}
