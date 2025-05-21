package core

type Hasher interface {
	Sum64(string) uint64
}

type Hasher32 interface {
	Sum32(string) uint32
}

func NewDefaultHasher32() Hasher32 {
	return fnv32a{}
}

// newDefaultHasher returns a new 64-bit FNV-1a Hasher which makes no memory allocations.
// Its Sum64 method will lay the value out in big-endian byte order.
// See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function
func newDefaultHasher() Hasher {
	return fnv64a{}
}

func NewDefaultHasher64() Hasher {
	return fnv64a{}
}

type fnv64a struct{}
type fnv32a struct{}

const (
	// offset64 FNVa offset basis. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	offset64 = 14695981039346656037
	// prime64 FNVa prime value. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	prime64 = 1099511628211

	offset32 = 2166136261
	prime32  = 16777619
)

// Sum64 gets the string and returns its uint64 hash value.
func (f fnv64a) Sum64(key string) uint64 {
	var hash uint64 = offset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}

	return hash
}

// Sum64 gets the string and returns its uint64 hash value.
func (f fnv32a) Sum32(key string) uint32 {
	var hash uint32 = offset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}

	return hash
}
