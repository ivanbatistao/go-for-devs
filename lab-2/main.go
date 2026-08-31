package main

import (
	"fmt"
	"lab-2/store"
)

func main() {
	s := store.NewMemoryStore()
	s.Set("name", "Ivan")

	value, ok := s.Get("name")

	fmt.Printf("Got value from storage: %s, (ok=%v)", value, ok)
}
