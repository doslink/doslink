package state

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	tm_db "github.com/tendermint/tmlibs/db"
)

func NewEvmDbWrapper(db tm_db.DB) *EvmDbWrapper {
	return &EvmDbWrapper{db: db}
}

type EvmDbWrapper struct {
	db tm_db.DB
}

func (db *EvmDbWrapper) Put(key []byte, value []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	db.db.Set(key, value)
	return
}

func (db *EvmDbWrapper) Get(key []byte) (val []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	val = db.db.Get(key)
	return
}

func (db *EvmDbWrapper) Has(key []byte) (has bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	has = db.db.Get(key) != nil
	return
}

func (db *EvmDbWrapper) Delete(key []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	db.db.Delete(key)
	return
}

func (db *EvmDbWrapper) Close() {
	db.db.Close()
}

func (db *EvmDbWrapper) NewBatch() ethdb.Batch {
	batch := db.db.NewBatch()
	return &batchWrapper{db: db, batch: batch}
}

type batchWrapper struct {
	db    *EvmDbWrapper
	batch tm_db.Batch
	size  int
}

func (b *batchWrapper) Put(key, value []byte) error {
	b.batch.Set(key, value)
	b.size += len(value)
	return nil
}

func (b *batchWrapper) Delete(key []byte) error {
	b.batch.Delete(key)
	b.size += 1
	return nil
}

func (b *batchWrapper) Write() error {
	b.batch.Write()
	return nil
}

func (b *batchWrapper) ValueSize() int {
	return b.size
}

func (b *batchWrapper) Reset() {
	b.batch = b.db.db.NewBatch()
	b.size = 0
}

//func (b *batchWrapper) Set(key, value []byte) {
//	b.batch.Put(key, value)
//}
//
//func (b *batchWrapper) Delete(key []byte) {
//	b.batch.Delete(key)
//}
//
//func (b *batchWrapper) Write() {
//	err := b.db.db.DB().Write(b.batch, nil)
//	if err != nil {
//		PanicCrisis(err)
//	}
//}
