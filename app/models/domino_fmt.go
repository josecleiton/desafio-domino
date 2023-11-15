package models

import "fmt"

const (
	dominoUnicodeHorizontal = '🀱'
	dominoUnicodeVertical   = '🁣'
)

func (d Domino) String() string {
	offset := d.X*DominoUniqueBones + d.Y

	base := int(dominoUnicodeHorizontal)
	if d.IsDouble() {
		base = dominoUnicodeVertical
	}

	return fmt.Sprintf("%c", base+offset)
}
