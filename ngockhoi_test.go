package tytd_go

import "testing"

func TestNgocKhoiEnterPayload(t *testing.T) {
	got := ngocKhoiEnterPayload(1, 9064454)
	want := []byte{0x08, 0xc1, 0x02, 0x10, 0x01, 0x18, 0x86, 0xa0, 0xa9, 0x04}
	if string(got) != string(want) {
		t.Fatalf("payload=%x want=%x", got, want)
	}
}
