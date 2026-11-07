// Package cache provides the token/data cache contract and an in-process
// implementation used across atto-wechat.
//
// It is part of Tier 0: it depends only on the standard library. The Cache
// interface is intentionally small (PSR-16 shaped) so that the bundled
// MemoryCache can later be swapped for a distributed backend (e.g. Redis)
// without touching callers such as the auth token manager.
//
// Operations that cannot be expressed in that shape -- ones that must be atomic
// to be correct -- live in optional capability interfaces instead of being added
// to Cache. A backend that cannot implement them atomically stays usable for
// token caching, where last-write-wins is harmless. See SetNXer.
package cache

import "time"

// Cache is the token/data cache contract used by auth and other modules.
//
// Keys are strings; values are raw bytes so any serializable payload (a token
// string, a JSON document, ...) can be stored. A ttl of 0 means "never
// expires"; a negative ttl expires the entry immediately (equivalent to
// Delete).
type Cache interface {
	// Get returns the value for key and whether it was found and unexpired.
	// A missing or expired entry yields (nil, false).
	Get(key string) ([]byte, bool)
	// Set stores value under key with the given time-to-live.
	Set(key string, value []byte, ttl time.Duration)
	// Delete removes key. Deleting a missing key is a no-op.
	Delete(key string)
	// Has reports whether key exists and is unexpired.
	Has(key string) bool
}

// SetNXer is an optional Cache capability: an atomic "store only if absent".
//
// It exists because Get followed by Set is not a substitute. Two processes can
// both observe a miss and both write, after which both believe they were first.
// That is exactly the race callback deduplication has to resolve, so a
// Deduplicator built on a cache needs this guarantee rather than a
// read-then-write.
//
// A distributed backend implements it in one round trip (Redis SET NX PX).
// MemoryCache implements it under its existing mutex.
//
// It is a separate interface rather than a Cache method so that implementing
// Cache does not force a backend to support atomics it cannot provide.
type SetNXer interface {
	// SetNX stores value under key with ttl only if key is absent or already
	// expired, reporting whether it stored.
	//
	// A ttl of zero or less means the entry never expires: SetNX has no
	// "expire immediately" meaning, because storing and then discarding would
	// always report success and deduplicate nothing.
	//
	// The error is returned rather than folded into the bool so that a caller
	// can tell "the key was already there" from "the store is unreachable".
	// Those two demand opposite reactions: the first suppresses work, the second
	// must not.
	SetNX(key string, value []byte, ttl time.Duration) (bool, error)
}

// compile-time guarantee that MemoryCache satisfies Cache.
var _ Cache = (*MemoryCache)(nil)

// compile-time guarantee that MemoryCache satisfies SetNXer.
var _ SetNXer = (*MemoryCache)(nil)
