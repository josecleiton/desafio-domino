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

func (l LinkedList[T]) HeadSafe() *Node[T] {
	if l.Len() < 1 {
		return nil
	}

	node := new(Node[T])
	*node = *l.head

	if node.Next == nil {
		node.Next = node
	}
	if node.Prev == nil {
		node.Prev = node
	}

	return node
}

func (l LinkedList[T]) Tail() *Node[T] {
	if l.Len() < 1 {
		return nil
	}

	return l.tail
}

func (l *LinkedList[T]) incLength() {
	l.length++
}

func (l *LinkedList[T]) Push(data *T) (*LinkedList[T], int) {
	defer l.incLength()
	if l.head == nil {
		l.head = new(Node[T])
		l.tail = l.head

		*l.head = Node[T]{Data: data}
		return l, l.Len()
	}

	l.tail.Next = new(Node[T])
	*l.tail.Next = Node[T]{Data: data, Prev: l.tail}
	l.tail = l.tail.Next

	return l, l.Len()
}

func (l *LinkedList[T]) PopFront() (*T, int) {
	if l.head == nil {
		return nil, l.Len()
	}

	if l.head == l.tail {
		l.tail = nil
	}

	nodeToPop := l.head

	l.head = nodeToPop.Next
	l.length--

	return nodeToPop.Data, l.Len()
}

func (l *LinkedList[T]) Clear() {
	for l.Len() > 0 {
		l.PopFront()
	}
}

func (l LinkedList[T]) Slice() []*T {
	slice := make([]*T, 0, l.Len())

	for node := l.head; node != nil; node = node.Next {
		slice = append(slice, node.Data)
	}

	return slice
}

func NewLinkedList[T interface{}]() *LinkedList[T] {
	return new(LinkedList[T])
}
