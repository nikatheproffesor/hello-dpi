package divert

import "testing"

func TestDivertStatus(t *testing.T) {
	st := GetStatus()
	// Status should return without panic
	if st.Active && !st.Supported {
		t.Errorf("divert cannot be active if not supported")
	}
}
