package y_crdt

type ID struct {
	AbstractType
	Client Number // client ID
	Clock  Number // unique per client id, continuous number
}

// GenID generates a new ID with the given client and clock values.
func GenID(client Number, clock Number) ID {
	return ID{
		Client: client,
		Clock:  clock,
	}
}

// CompareIDs compares two IDs for equality.
func CompareIDs(a *ID, b *ID) bool {
	return a == b || (a != nil && b != nil && a.Client == b.Client && a.Clock == b.Clock)
}

