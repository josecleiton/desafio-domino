package models

import "fmt"

const (
	dominoUnicodeHorizontal = '🀱'
	dominoUnicodeVertical   = '🁣'
	thinSpace               = ' '
)

func (d Domino) String() string {
	offset := d.L*DominoUniqueBones + d.R

	base := int(dominoUnicodeHorizontal)
	if d.IsDouble() {
		base = dominoUnicodeVertical
	}

	return fmt.Sprintf("%c", base+offset)
}

func (e Edges) String() string {
	result := ""
	for k, v := range e {
		if v != nil {
			result += fmt.Sprintf("{%v, %v} ", k, v)
		}
	}

	return result
}
