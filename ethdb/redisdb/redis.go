package redisdb

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/redis/go-redis/v9"
)

type simpleClient interface {
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	MGet(ctx context.Context, keys ...string) *redis.SliceCmd
	MSet(ctx context.Context, pairs ...interface{}) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	ConfigGet(ctx context.Context, parameter string) *redis.MapStringStringCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	Pipeline() redis.Pipeliner
	Close() error
}

// Database is a key-value lookup for redis.
type Database struct {
	client simpleClient
	log    log.Logger // Contextual logger tracking the database endpoint
}

// key convert slice to string
func (db *Database) key(key []byte) string {
	return string(key)
}

func (db *Database) Has(key []byte) (bool, error) {
	val, err := db.client.Exists(context.Background(), db.key(key)).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (db *Database) Get(key []byte) ([]byte, error) {
	val, err := db.client.Get(context.Background(), db.key(key)).Result()
	if err != nil {
		if err == redis.Nil {
			//TODO return nil or empty slice?
			return nil, nil
		}
		return nil, err
	}

	return []byte(val), nil
}

func (db *Database) Put(key []byte, value []byte) error {
	return db.client.Set(context.Background(), db.key(key), value, time.Duration(0)).Err()
}

func (db *Database) Delete(key []byte) error {
	return db.client.Del(context.Background(), db.key(key)).Err()
}

func (db *Database) Close() error {
	return db.client.Close()
}

func (db *Database) Stat(property string) (string, error) {
	config := db.client.ConfigGet(context.Background(), property)
	if err := config.Err(); err != nil {
		return "", err
	}
	return config.String(), nil
}

func (db *Database) NewBatch() ethdb.Batch {
	return newBatch(db, 0)
}

func (db *Database) NewBatchWithSize(size int) ethdb.Batch {
	return newBatch(db, size)
}

func (db *Database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return newIterator(db, prefix, start)
}

func (db *Database) Compact(start []byte, limit []byte) error {
	//Does nothing
	return nil
}

func (db *Database) NewSnapshot() (ethdb.Snapshot, error) {
	return newSnapshot(db)
}

func NewRedisDB(endpoint, password string) (*Database, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     endpoint,
		Password: password, // no password set
		DB:       0,        // use default DB
	})
	logger := log.New("endpoint", endpoint)
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return &Database{rdb, logger}, nil
}
