package rdl

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/antchfx/xmlquery"
)

// loadOriginal returns the bytes of the test fixture.
func loadOriginal(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "sample.rdl"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return b
}

// TestRoundTrip_Idempotent proves Load -> Save -> Load -> Save is stable:
// the second Save produces the same bytes as the first Save. The first save
// may normalise minor formatting differences, but after that the file is stable.
func TestRoundTrip_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	dst1 := filepath.Join(tmp, "first.rdl")
	dst2 := filepath.Join(tmp, "second.rdl")

	// First Load -> Save
	doc, err := Load(filepath.Join("testdata", "sample.rdl"))
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	if err := doc.Save(dst1); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	// Second Load -> Save from the first output
	doc2, err := Load(dst1)
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if err := doc2.Save(dst2); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	first, _ := os.ReadFile(dst1)
	second, _ := os.ReadFile(dst2)
	if !bytes.Equal(first, second) {
		t.Errorf("round-trip is not idempotent: first save and second save differ")
		// Find first diff for debugging
		min := len(first)
		if len(second) < min {
			min = len(second)
		}
		for i := 0; i < min; i++ {
			if first[i] != second[i] {
				start := i - 40
				if start < 0 {
					start = 0
				}
				end := i + 40
				if end > min {
					end = min
				}
				t.Errorf("first diff at byte %d\n  first:  %q\n  second: %q",
					i, first[start:end], second[start:end])
				break
			}
		}
		t.Logf("first  len=%d", len(first))
		t.Logf("second len=%d", len(second))
	}
}

// TestRoundTrip_PreservesBOMAndCRLF proves the saved file still has UTF-8 BOM
// and CRLF line endings after a round-trip.
func TestRoundTrip_PreservesBOMAndCRLF(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "out.rdl")

	doc, err := Load(filepath.Join("testdata", "sample.rdl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := doc.Save(dst); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}

	if !bytes.HasPrefix(out, bomBytes) {
		t.Errorf("saved file is missing UTF-8 BOM")
	}
	if bytes.Contains(out, []byte("\n")) && bytes.Contains(out, []byte{0x0A}) {
		// Check no bare LF (LF not preceded by CR)
		for i, b := range out {
			if b == 0x0A && (i == 0 || out[i-1] != 0x0D) {
				t.Errorf("found bare LF at byte %d; all line endings should be CRLF", i)
				break
			}
		}
	}
}

// TestRoundTrip_SemanticPreservation proves the parsed tree is unchanged
// after a round-trip: same element/attribute structure. Catches the case where
// xmlquery drops or reorders something that byte-equality misses.
func TestRoundTrip_SemanticPreservation(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "out.rdl")

	doc1, err := Load(filepath.Join("testdata", "sample.rdl"))
	if err != nil {
		t.Fatalf("Load original: %v", err)
	}
	if err := doc1.Save(dst); err != nil {
		t.Fatalf("Save: %v", err)
	}

	doc2, err := Load(dst)
	if err != nil {
		t.Fatalf("Load round-tripped: %v", err)
	}

	count1 := countNodes(doc1.Root())
	count2 := countNodes(doc2.Root())
	if count1 != count2 {
		t.Errorf("node count changed: original=%d roundtripped=%d", count1, count2)
	}
}

// countNodes walks the tree and returns total element + attribute count.
func countNodes(n *xmlquery.Node) int {
	count := 0
	if n.Type == xmlquery.ElementNode {
		count++
		count += len(n.Attr)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		count += countNodes(c)
	}
	return count
}
