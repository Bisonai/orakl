package set

type Set[T comparable] struct {
	data map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{data: make(map[T]struct{})}
}

func (s *Set[T]) Add(element T) {
	s.data[element] = struct{}{}
}

func (s *Set[T]) Remove(element T) {
	delete(s.data, element)
}

func (s *Set[T]) Contains(element T) bool {
	_, exists := s.data[element]
	return exists
}

func (s *Set[T]) Size() int {
	return len(s.data)
}
