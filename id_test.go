package y_crdt

import "testing"

func TestGenID(t *testing.T) {
	id := GenID(1, 2)
	if id.Client != 1 {
		t.Errorf("id.Client = %d, want 1", id.Client)
	}

	if id.Clock != 2 {
		t.Errorf("id.Clock = %d, want 2", id.Clock)
	}
}

func TestCompareIDs(t *testing.T) {
	id1 := GenID(1, 2)
	id2 := GenID(1, 3)
	id3 := GenID(1, 2)
	if CompareIDs(&id1, &id2) {
		t.Error("CompareIDs(id1, id2) = true, want false")
	}

	if !CompareIDs(&id1, &id3) {
		t.Error("CompareIDs(id1, id3) = false, want true")
	}
}
