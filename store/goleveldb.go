package store

import (
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

type GolevelDB struct {
	db *leveldb.DB
}

var _ KVStore = &GolevelDB{}
var _ Batch = &golevelDBBatch{}

func NewGoLevelDB(name string, dir string) (*GolevelDB, error) {
	return NewGoLevelDBWithOpts(name, dir, nil)
}

func NewGoLevelDBWithOpts(name string, dir string, o *opt.Options) (*GolevelDB, error) {
	dbPath := filepath.Join(dir, name+".db")
	db, err := leveldb.OpenFile(dbPath, o)
	if err != nil {
		fmt.Printf("NewGoLevelDBWithOpts err=%v\n", err.Error())
		return nil, err
	}

	fmt.Printf("NewGoLevelDBWithOpts opened")
	database := &GolevelDB{
		db: db,
	}
	return database, nil
}

func (g *GolevelDB) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrKeyEmpty
	}

	res, err := g.db.Get(key, nil)
	if err != nil {
		if err == ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *GolevelDB) Set(key []byte, value []byte) error {
	if len(key) == 0 {
		return ErrKeyEmpty
	}
	if len(value) == 0 {
		return ErrValueNil
	}

	if err := g.db.Put(key, value, nil); err != nil {
		return err
	}
	return nil
}

func (g *GolevelDB) Delete(key []byte) error {
	if len(key) == 0 {
		return ErrKeyEmpty
	}

	if err := g.db.Delete(key, nil); err != nil {
		return err
	}
	return nil
}

type golevelDBBatch struct {
	db    *GolevelDB
	batch *leveldb.Batch
}

func (g *golevelDBBatch) Set(key, value []byte) error {
	if len(key) == 0 {
		return ErrKeyEmpty
	}
	if len(value) == 0 {
		return ErrValueNil
	}
	if g.batch == nil {
		return ErrBatchClosed
	}

	g.batch.Put(key, value)
	return nil
}

func (g *golevelDBBatch) Delete(key []byte) error {
	if len(key) == 0 {
		return ErrKeyEmpty
	}
	g.batch.Delete(key)
	return nil
}

func (g *golevelDBBatch) Commit() error {
	return nil
}

func (g *golevelDBBatch) Discard() {
}

func newGoLevelDBBatch() *golevelDBBatch {
	return &golevelDBBatch{
		batch: new(leveldb.Batch),
	}
}

func (g *GolevelDB) NewBatch() Batch {
	return newGoLevelDBBatch()
}

type golevelDBIterator struct {
	prefix, start, end   []byte
	source               iterator.Iterator
	isReverse, isInvalid bool
}

func newGolevelDBIterator(source iterator.Iterator, start, end []byte, isReverse bool) *golevelDBIterator {
	if isReverse {
		if end == nil {
			source.Last()
		} else {
			valid := source.Seek(end)
			if valid {
				eoakey := source.Key() // end or after key
				if bytes.Compare(end, eoakey) <= 0 {
					source.Prev()
				}
			} else {
				source.Last()
			}
		}
	} else {
		if start == nil {
			source.First()
		} else {
			source.Seek(start)
		}
	}
	return &golevelDBIterator{
		source:    source,
		start:     start,
		end:       end,
		isReverse: isReverse,
		isInvalid: false,
	}
}

func (g *golevelDBIterator) Valid() bool {
	// Once invalid, forever invalid.
	if g.isInvalid {
		return false
	}

	// If source errors, invalid.
	if err := g.Error(); err != nil {
		g.isInvalid = true
		return false
	}

	// If source is invalid, invalid.
	if !g.source.Valid() {
		g.isInvalid = true
		return false
	}

	// If key is end or past it, invalid.
	start := g.start
	end := g.end
	key := g.source.Key()

	if g.isReverse {
		if start != nil && bytes.Compare(key, start) < 0 {
			g.isInvalid = true
			return false
		}
	} else {
		if end != nil && bytes.Compare(end, key) <= 0 {
			g.isInvalid = true
			return false
		}
	}

	// Valid
	return true
}

func (g *golevelDBIterator) Next() {
	g.assertIsValid()
	if g.isReverse {
		g.source.Prev()
	} else {
		g.source.Next()
	}
}

func (g *golevelDBIterator) Key() []byte {
	// Key returns a copy of the current key.
	// See https://github.com/syndtr/goleveldb/blob/52c212e6c196a1404ea59592d3f1c227c9f034b2/leveldb/iterator/iter.go#L88
	g.assertIsValid()
	return cp(g.source.Key())
}

func (g *golevelDBIterator) Value() []byte {
	// Value returns a copy of the current value.
	// See https://github.com/syndtr/goleveldb/blob/52c212e6c196a1404ea59592d3f1c227c9f034b2/leveldb/iterator/iter.go#L88
	g.assertIsValid()
	return cp(g.source.Value())
}

func (g *golevelDBIterator) Error() error {
	return g.source.Error()
}

func (g *golevelDBIterator) Discard() {
}

var _ Iterator = &golevelDBIterator{}

func (g *GolevelDB) PrefixIterator(prefix []byte) Iterator {
	slice := util.BytesPrefix(prefix)
	iter := g.db.NewIterator(slice, nil)
	return newGolevelDBIterator(iter, slice.Start, slice.Limit, false)
}

func (g *golevelDBIterator) assertIsValid() {
	if !g.Valid() {
		panic("iterator is invalid")
	}
}

func cp(bz []byte) (ret []byte) {
	ret = make([]byte, len(bz))
	copy(ret, bz)
	return ret
}
