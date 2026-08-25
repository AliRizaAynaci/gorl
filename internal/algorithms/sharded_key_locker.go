package algorithms

import (
	"hash/maphash"
	"sync"
)

const keyLockShardCount = 256

// shardedKeyLocker bounds lock memory while allowing unrelated keys to make
// progress independently. Hash collisions may still serialize two keys, but
// the randomized seed prevents callers from deliberately targeting a shard.
//
// Every generic stateful limiter allocates one unconditionally. The whole array
// is ~2 KB and does not grow with key cardinality, which is a smaller price than
// keeping "this limiter has no locker" as an invariant each call site must
// re-derive before touching it.
type shardedKeyLocker struct {
	seed   maphash.Seed
	shards [keyLockShardCount]sync.Mutex
}

func newShardedKeyLocker() *shardedKeyLocker {
	return &shardedKeyLocker{seed: maphash.MakeSeed()}
}

func (l *shardedKeyLocker) mutexFor(key string) *sync.Mutex {
	index := maphash.String(l.seed, key) & (keyLockShardCount - 1)
	return &l.shards[index]
}
