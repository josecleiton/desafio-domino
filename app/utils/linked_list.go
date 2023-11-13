package utils

type Node[T interface{}] struct {
	Data *T
	Prev *Node[T]
	Next *Node[T]
}

type LinkedList[T interface{}] struct {
	head   *Node[T]
	tail   *Node[T]
	length int
}

func (l LinkedList[T]) Len() int {
	return l.length
}

func (l LinkedList[T]) Head() Node[T] {
	return *l.head
}

func (l LinkedList[T]) Tail() Node[T] {
	return *l.tail
}

func (l *LinkedList[T]) Push(data *T) (*LinkedList[T], int) {
	if l.head == nil {
		l.head = &Node[T]{Data: data}
		l.tail = l.head
	}

	l.tail.Next = &Node[T]{Data: data, Prev: l.tail}
	l.tail = l.tail.Next

	l.length++

	return l, l.Len()
}

func (l LinkedList[T]) Slice() []*T {
	slice := make([]*T, 0, l.Len())

	for node := l.head; node != nil; node = node.Next {
		slice = append(slice, node.Data)
	}

	return slice
}
