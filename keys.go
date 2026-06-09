package main

type keyKind int

const (
	kindChar keyKind = iota
	kindEnter
	kindBackspace
	kindCtrlC
	kindUp
	kindDown
	kindLeft
	kindRight
	kindEsc
)

type keyEvent struct {
	kind keyKind
	ch   byte
}

// parseKeys decodes a chunk of raw terminal input into key events.
func parseKeys(b []byte, emit func(keyEvent)) {
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == 0x1b && i+2 < len(b) && b[i+1] == '[':
			switch b[i+2] {
			case 'A':
				emit(keyEvent{kind: kindUp})
			case 'B':
				emit(keyEvent{kind: kindDown})
			case 'C':
				emit(keyEvent{kind: kindRight})
			case 'D':
				emit(keyEvent{kind: kindLeft})
			}
			i += 2
		case c == 0x1b:
			emit(keyEvent{kind: kindEsc})
		case c == 0x03:
			emit(keyEvent{kind: kindCtrlC})
		case c == '\r' || c == '\n':
			emit(keyEvent{kind: kindEnter})
		case c == 0x7f || c == 0x08:
			emit(keyEvent{kind: kindBackspace})
		case c >= 0x20 && c < 0x7f:
			emit(keyEvent{kind: kindChar, ch: c})
		}
	}
}
