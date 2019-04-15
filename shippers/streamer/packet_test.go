package streamer

import "testing"

func TestPacket(t *testing.T) {
	pkt := newPacket(1300)
	testBytes := []byte("abcdefghijklmnopqrstuvwxyz")
	pkt.add([]byte(testBytes))
	output := pkt.read()

	if output[0] != testBytes[0] {
		t.Logf("packet buffer did not return the correct bytes")
		t.Fail()
	}
	if len(output) != len(testBytes) {
		t.Logf("packet buffer didn't return the correct number of bytes. Want: %d, Got: %d", len(testBytes), len(output))
	}
}
