package a

import "testing"

// Reported: a name, and nothing behind it.
func TestSnapshotRead(t *testing.T) {
	t.Skip("requires a live database; see the integration tests") // want `TestSnapshotRead: test body is only a skip`
}

// Every skip spelling is the same defect.
func TestSkipf(t *testing.T) {
	t.Skipf("later %d", 1) // want `TestSkipf: test body is only a skip`
}

func TestSkipNow(t *testing.T) {
	t.SkipNow() // want `TestSkipNow: test body is only a skip`
}

// An unconditional subtest stub is as dead as an unconditional test.
func TestGapClosure(t *testing.T) {
	t.Run("designer", func(t *testing.T) {
		t.Skip("source has moved") // want `designer: test body is only a skip`
	})
}

// A subtest with a computed name is still reported.
func TestComputedSubtestName(t *testing.T) {
	name := "x"
	t.Run(name, func(t *testing.T) {
		t.Skip("todo") // want `subtest: test body is only a skip`
	})
}

// A bare marker is itself a finding: the exemption has to say why.
func TestBareMarker(t *testing.T) {
	//emptyskip:ok
	t.Skip("later") // want `TestBareMarker: //emptyskip:ok needs a reason`
}

// Not reported: a skip with real statements behind it is pending work.
func TestKnownOpenBug(t *testing.T) {
	t.Skip("open bug: CHECK violation reports the wrong code")
	rows := query()
	if len(rows) != 1 {
		t.Fatalf("want 1 row")
	}
}

// Not reported: the standard environment gate.
func TestNeedsNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	t.Log("real work")
}

// Not reported: the per-case N/A shape. The subtest exists only for the cases
// it names.
func TestCursorContract(t *testing.T) {
	for _, tc := range cases() {
		if tc.naReason != "" {
			t.Run("has_more__NA", func(t *testing.T) {
				t.Skipf("N/A: %s", tc.naReason)
			})
		}
	}
}

// Not reported: the same N/A shape inside a switch.
func TestCursorContractSwitch(t *testing.T) {
	for _, tc := range cases() {
		switch tc.naReason {
		case "":
		default:
			t.Run("na", func(t *testing.T) {
				t.Skip(tc.naReason)
			})
		}
	}
}

// Not reported: the marker with a reason, for a behaviour that cannot be
// tested rather than merely is not.
func TestUnreachableGrammar(t *testing.T) {
	//emptyskip:ok — this shape is not constructible from source text
	t.Skip("not reachable from this grammar")
}

// Not reported: a helper whose body is a skip is not a test.
func helperSkip(t *testing.T) {
	t.Skip("helper")
}

// Not reported: a Test-prefixed func that is not a test signature.
func TestdataPath(t *testing.T, dir string) {
	t.Skip(dir)
}

// Not reported: Skip on a type outside package testing.
type skipper struct{}

func (skipper) Skip(...any) {}

func TestOtherSkip(t *testing.T) {
	skipper{}.Skip("not testing's skip")
}

type c struct{ naReason string }

func cases() []c   { return nil }
func query() []int { return nil }

var _ = helperSkip
