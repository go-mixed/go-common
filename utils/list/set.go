package list_utils

import (
	"fmt"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
)

// Set - uses map as set of strings.
type Set[T comparable] map[T]struct{}

// NewSet - creates new string set.
func NewSet[T comparable](sl ...T) Set[T] {
	set := make(Set[T])
	for _, k := range sl {
		set.Add(k)
	}
	return set
}

// ToSlice - returns Set as string slice.
func (s Set[T]) ToSlice() []T {
	keys := make([]T, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

// IsEmpty - returns whether the set is empty or not.
func (s Set[T]) IsEmpty() bool {
	return len(s) == 0
}

// Add - adds string to the set.
func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

// Remove - removes string in the set.  It does nothing if string does not exist in the set.
func (s Set[T]) Remove(item T) {
	delete(s, item)
}

// Contains - checks if string is in the set.
func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

// Match - returns new set containing each value who passes match function.
// A 'matchFn' should accept element in a set as first argument and
// 'matchString' as second argument.  The function can do any logic to
// compare both the arguments and should return true to accept element in
// a set to include in output set else the element is ignored.
func (s Set[T]) Match(item T, matchFn func(T, T) bool) Set[T] {
	nset := NewSet[T]()
	for k := range s {
		if matchFn(k, item) {
			nset.Add(k)
		}
	}
	return nset
}

// ApplyFunc - returns new set containing each value processed by 'applyFn'.
// A 'applyFn' should accept element in a set as a argument and return
// a processed string.  The function can do any logic to return a processed
// string.
func (s Set[T]) ApplyFunc(applyFn func(T) T) Set[T] {
	nset := NewSet[T]()
	for k := range s {
		nset.Add(applyFn(k))
	}
	return nset
}

// Equals - checks whether given set is equal to current set or not.
func (s Set[T]) Equals(sset Set[T]) bool {
	// If length of s is not equal to length of given s, the
	// s is not equal to given s.
	if len(s) != len(sset) {
		return false
	}

	// As both sets are equal in length, check each element are equal.
	for k := range s {
		if _, ok := sset[k]; !ok {
			return false
		}
	}

	return true
}

// Intersection - returns the intersection with given set as new set.
func (s Set[T]) Intersection(sset Set[T]) Set[T] {
	nset := NewSet[T]()
	for k := range s {
		if _, ok := sset[k]; ok {
			nset.Add(k)
		}
	}

	return nset
}

// Difference - returns the difference with given set as new set.
func (s Set[T]) Difference(sset Set[T]) Set[T] {
	nset := NewSet[T]()
	for k := range s {
		if _, ok := sset[k]; !ok {
			nset.Add(k)
		}
	}

	return nset
}

// Union - returns the union with given set as new set.
func (s Set[T]) Union(sset Set[T]) Set[T] {
	nset := NewSet[T]()
	for k := range s {
		nset.Add(k)
	}

	for k := range sset {
		nset.Add(k)
	}

	return nset
}

// MarshalJSON - converts to JSON data.
func (s Set[T]) MarshalJSON() ([]byte, error) {
	return text_utils.JsonMarshalToBytes(s.ToSlice())
}

// UnmarshalJSON - parses JSON data and creates new set with it.
// If 'data' contains JSON string array, the set contains each string.
// If 'data' contains JSON string, the set contains the string as one element.
// If 'data' contains Other JSON types, JSON parse error is returned.
func (s Set[T]) UnmarshalJSON(data []byte) error {
	var sl []T
	err := text_utils.JsonUnmarshalFromBytes(data, &sl)
	if err == nil {
		for _, item := range sl {
			s.Add(item)
		}
	}

	return err
}

// String - returns printable string of the set.
func (s Set[T]) String() string {
	return fmt.Sprintf("%s", s.ToSlice())
}

// Clone - returns copy of given set.
func (s Set[T]) Clone() Set[T] {
	nset := NewSet[T]()
	for k, v := range s {
		nset[k] = v
	}
	return nset
}
