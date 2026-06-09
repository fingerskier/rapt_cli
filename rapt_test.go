package main

import (
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	st := loadState()
	if st.WPM != 0 || len(st.Files) != 0 {
		t.Fatalf("expected empty state, got %+v", st)
	}

	st.WPM = 425
	st.LastFile = "/tmp/book.txt"
	st.Files["/tmp/book.txt"] = 1234
	if err := st.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := loadState()
	if got.WPM != 425 || got.LastFile != "/tmp/book.txt" || got.Files["/tmp/book.txt"] != 1234 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestParseKeys(t *testing.T) {
	cases := []struct {
		in   []byte
		want []keyEvent
	}{
		{[]byte("q"), []keyEvent{{kind: kindChar, ch: 'q'}}},
		{[]byte{0x03}, []keyEvent{{kind: kindCtrlC}}},
		{[]byte("\x1b[C"), []keyEvent{{kind: kindRight}}},
		{[]byte("\x1b[D\x1b[A"), []keyEvent{{kind: kindLeft}, {kind: kindUp}}},
		{[]byte("\r"), []keyEvent{{kind: kindEnter}}},
		{[]byte{0x7f}, []keyEvent{{kind: kindBackspace}}},
		{[]byte("ab"), []keyEvent{{kind: kindChar, ch: 'a'}, {kind: kindChar, ch: 'b'}}},
	}
	for _, c := range cases {
		var got []keyEvent
		parseKeys(c.in, func(ev keyEvent) { got = append(got, ev) })
		if len(got) != len(c.want) {
			t.Fatalf("parseKeys(%q) = %v, want %v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseKeys(%q)[%d] = %v, want %v", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestInterval(t *testing.T) {
	a := &app{wpm: 300}
	if got := a.interval(); got != 200*time.Millisecond {
		t.Errorf("interval at 300 wpm = %v, want 200ms", got)
	}
}

func TestSetWPMClamps(t *testing.T) {
	a := &app{wpm: defaultWPM}
	a.setWPM(10)
	if a.wpm != minWPM {
		t.Errorf("wpm = %d, want %d", a.wpm, minWPM)
	}
	a.setWPM(99999)
	if a.wpm != maxWPM {
		t.Errorf("wpm = %d, want %d", a.wpm, maxWPM)
	}
}

func TestStepClamps(t *testing.T) {
	a := &app{words: []string{"a", "b", "c"}, idx: 0}
	a.step(-1)
	if a.idx != 0 {
		t.Errorf("idx = %d, want 0", a.idx)
	}
	a.step(5)
	if a.idx != 2 {
		t.Errorf("idx = %d, want 2", a.idx)
	}
}
