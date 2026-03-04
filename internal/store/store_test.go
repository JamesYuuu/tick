package store

import "testing"

func TestStore_Interface(t *testing.T) {
	var _ Store = (*SQLiteStore)(nil)
}
