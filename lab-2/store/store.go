package store

type Store interface {
	Set(key string, value string)
	Get(key string) (string, bool)
	Delete(key string)
}

type MemoryStore struct {
	data map[string]string
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]string),
	}
}

func (store *MemoryStore) Set(key, value string) {
	store.data[key] = value
}

func (store *MemoryStore) Get(key string) (string, bool) {
	value, ok := store.data[key]

	return value, ok
}

func (store *MemoryStore) Delete(key string) {
	delete(store.data, key)
}
