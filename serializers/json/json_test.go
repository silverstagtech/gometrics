package json

import "testing"

func TestBuildName(t *testing.T) {
	out := buildName("test", "name")
	expected := "test_name"
	if out != expected {
		t.Logf("buildName did not give expected value. Want: %s, Got: %s", expected, out)
		t.Fail()
	}
}
