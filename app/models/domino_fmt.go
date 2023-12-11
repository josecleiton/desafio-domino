package models

import (
	"fmt"
	"strings"
)

const (
	dominoUnicodeHorizontal = '🀱'
	dominoUnicodeVertical   = '🁣'
	hairSpace               = ' '
)

func (d Domino) String() string {
	offset := d.L*DominoUniqueBones + d.R

	space := hairSpace
	base := int(dominoUnicodeHorizontal)
	if d.IsDouble() {
		base = dominoUnicodeVertical
		space = 0
	}

	return fmt.Sprintf("%c%c", base+offset, space)
}

func (e Edges) String() string {
	builder := strings.Builder{}
	for k, v := range e {
		if v != nil {
			builder.WriteString(fmt.Sprintf("{%v, %v} ", k, v))
		}
	}

	return builder.String()
}

func TableString(table []Domino) string {
	builder := strings.Builder{}

	builder.WriteRune('[')

	for _, d := range table {
		builder.WriteString(d.String())
	}

	builder.WriteRune(']')

	return builder.String()
}

func (p DominoPlay) String() string {
	return fmt.Sprintf(
		"{Player: %d, Bone: %s}",
		p.PlayerPosition,
		p.Bone,
	)
}
