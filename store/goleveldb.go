package store

import (
	"errors"
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
		return nil, err
	}

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
		if errors.Is(err, leveldb.ErrNotFound) {
			return nil, ErrKeyNotFound
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
	if g.batch == nil {
		return ErrBatchClosed
	}
	err := g.db.db.Write(g.batch, nil)
	if err != nil {
		return err
	}
	return g.Close()
}

func (g *golevelDBBatch) Close() error {
	if g.batch != nil {
		g.batch.Reset()
		g.batch = nil
	}
	return nil
}

func (g *golevelDBBatch) Discard() {
}

func newGoLevelDBBatch(db *GolevelDB) *golevelDBBatch {
	return &golevelDBBatch{
		db:    db,
		batch: new(leveldb.Batch),
	}
}

func (g *GolevelDB) NewBatch() Batch {
	return newGoLevelDBBatch(g)
}

type golevelDBIterator struct {
	prefix []byte
	source iterator.Iterator
}

func newGolevelDBIterator(source iterator.Iterator, prefix []byte) *golevelDBIterator {
	source.Seek(prefix)
	return &golevelDBIterator{
		source: source,
		prefix: prefix,
	}
}

func (g *golevelDBIterator) Valid() bool {
	return g.source.Valid()
}

func (g *golevelDBIterator) Next() {
	g.assertIsValid()
	g.source.Next()
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
	return newGolevelDBIterator(iter, prefix)
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
