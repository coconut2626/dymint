package store

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewGolevelDB(t *testing.T) {
	name := fmt.Sprintf("test_%x", randStr(12))
	defer cleanupDBDir("", name)

	// Test we can't open the db twice for writing
	db, err := NewGoLevelDB(name, "")
	require.Nil(t, err)

	err = db.Set([]byte("aa"), []byte("bb"))
	require.Nil(t, err)
	val, err := db.Get([]byte("aa"))
	require.Nil(t, err)
	t.Log(string(val))
	require.Nil(t, err)

	err = db.Delete([]byte("abaaa"))
	require.Nil(t, err)

	_, err = NewGoLevelDB(name, "")
	require.NotNil(t, err)
}

func TestGoLevelDBGetErrors(t *testing.T) {
	name := fmt.Sprintf("test_%x", randStr(12))
	defer cleanupDBDir("", name)

	// Test we can't open the db twice for writing
	db, err := NewGoLevelDB(name, "")
	require.Nil(t, err)

	tc := []struct {
		name string
		key  []byte
		err  error
	}{
		{"empty key", []byte{}, ErrKeyEmpty},
		{"not found key", []byte("missing key"), ErrKeyNotFound},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Get(tt.key)
			if !errors.Is(err, tt.err) {
				t.Errorf("Invalid err, got: %v expected %v", err, tt.err)
			}
		})
	}
}

func TestGoLevelDBSetErrors(t *testing.T) {
	name := fmt.Sprintf("test_%x", randStr(12))
	defer cleanupDBDir("", name)

	// Test we can't open the db twice for writing
	db, err := NewGoLevelDB(name, "")
	require.Nil(t, err)

	tc := []struct {
		name  string
		key   []byte
		value []byte
		err   error
	}{
		{"empty key", []byte{}, []byte{}, ErrKeyEmpty},
		{"invalid key", []byte("!badger!key"), []byte("invalid header"), nil},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Set(tt.key, tt.value)
			if !errors.Is(tt.err, err) {
				t.Errorf("Invalid err, got: %v expected %v", err, tt.err)
			}
		})
	}
}

func TestGoLevelDBDeleteErrors(t *testing.T) {
	name := fmt.Sprintf("test_%x", randStr(12))
	defer cleanupDBDir("", name)

	// Test we can't open the db twice for writing
	db, err := NewGoLevelDB(name, "")
	require.Nil(t, err)

	tc := []struct {
		name string
		key  []byte
		err  error
	}{
		{"empty key", []byte{}, ErrKeyEmpty},
		{"invalid key", []byte("!badger!key"), nil},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Delete(tt.key)
			if !errors.Is(err, tt.err) {
				t.Errorf("Invalid err, got: %v expected %v", err, tt.err)
			}
		})
	}
}
