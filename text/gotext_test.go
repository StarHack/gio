package text

import (
	"math"
	"reflect"
	"testing"

	nsareg "eliasnaur.com/font/noto/sans/arabic/regular"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"gioui.org/font/opentype"
	"gioui.org/io/system"
)

var english = system.Locale{
	Language:  "EN",
	Direction: system.LTR,
}

var arabic = system.Locale{
	Language:  "AR",
	Direction: system.RTL,
}

func testShaper(faces ...Face) *shaperImpl {
	shaper := shaperImpl{}
	for _, face := range faces {
		shaper.Load(FontFace{Face: face})
	}
	return &shaper
}

func TestEmptyString(t *testing.T) {
	ppem := fixed.I(200)
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)

	lines := shaper.LayoutRunes(Parameters{PxPerEm: ppem}, 0, 2000, english, []rune{})
	if len(lines.lines) == 0 {
		t.Fatalf("Layout returned no lines for empty string; expected 1")
	}
	l := lines.lines[0]
	exp := fixed.Rectangle26_6{
		Min: fixed.Point26_6{
			Y: fixed.Int26_6(-12094),
		},
		Max: fixed.Point26_6{
			Y: fixed.Int26_6(2700),
		},
	}
	if got := l.bounds; got != exp {
		t.Errorf("got bounds %+v for empty string; expected %+v", got, exp)
	}
}

func TestAlignWidth(t *testing.T) {
	lines := []line{
		{width: fixed.I(50)},
		{width: fixed.I(75)},
		{width: fixed.I(25)},
	}
	for _, minWidth := range []int{0, 50, 100} {
		width := alignWidth(minWidth, lines)
		if width < minWidth {
			t.Errorf("expected width >= %d, got %d", minWidth, width)
		}
	}
}

func TestShapingAlignWidth(t *testing.T) {
	ppem := fixed.I(10)
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)

	type testcase struct {
		name               string
		minWidth, maxWidth int
		expected           int
		str                string
	}
	for _, tc := range []testcase{
		{
			name:     "zero min",
			maxWidth: 100,
			str:      "a\nb\nc",
			expected: 22,
		},
		{
			name:     "min == max",
			minWidth: 100,
			maxWidth: 100,
			str:      "a\nb\nc",
			expected: 100,
		},
		{
			name:     "min < max",
			minWidth: 50,
			maxWidth: 100,
			str:      "a\nb\nc",
			expected: 50,
		},
		{
			name:     "min < max, text > min",
			minWidth: 50,
			maxWidth: 100,
			str:      "aphabetic\nb\nc",
			expected: 60,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines := shaper.LayoutString(Parameters{PxPerEm: ppem}, tc.minWidth, tc.maxWidth, english, tc.str)
			if lines.alignWidth != tc.expected {
				t.Errorf("expected line alignWidth to be %d, got %d", tc.expected, lines.alignWidth)
			}
		})
	}
}

// TestNewlineSynthesis ensures that the shaper correctly inserts synthetic glyphs
// representing newline runes.
func TestNewlineSynthesis(t *testing.T) {
	ppem := fixed.I(10)
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	shaper := testShaper(ltrFace, rtlFace)

	type testcase struct {
		name   string
		locale system.Locale
		txt    string
	}
	for _, tc := range []testcase{
		{
			name:   "ltr bidi newline in rtl segment",
			locale: english,
			txt:    "The quick سماء שלום لا fox تمط שלום\n",
		},
		{
			name:   "ltr bidi newline in ltr segment",
			locale: english,
			txt:    "The quick سماء שלום لا fox\n",
		},
		{
			name:   "rtl bidi newline in ltr segment",
			locale: arabic,
			txt:    "الحب سماء brown привет fox تمط jumps\n",
		},
		{
			name:   "rtl bidi newline in rtl segment",
			locale: arabic,
			txt:    "الحب سماء brown привет fox تمط\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			doc := shaper.LayoutRunes(Parameters{PxPerEm: ppem}, 0, 200, tc.locale, []rune(tc.txt))
			for lineIdx, line := range doc.lines {
				lastRunIdx := len(line.runs) - 1
				lastRun := line.runs[lastRunIdx]
				lastGlyphIdx := len(lastRun.Glyphs) - 1
				if lastRun.Direction.Progression() == system.TowardOrigin {
					lastGlyphIdx = 0
				}
				glyph := lastRun.Glyphs[lastGlyphIdx]
				if glyph.glyphCount != 0 {
					t.Errorf("expected synthetic newline on line %d, run %d, glyph %d", lineIdx, lastRunIdx, lastGlyphIdx)
				}
				for runIdx, run := range line.runs {
					for glyphIdx, glyph := range run.Glyphs {
						if runIdx == lastRunIdx && glyphIdx == lastGlyphIdx {
							continue
						}
						if glyph.glyphCount == 0 {
							t.Errorf("found invalid synthetic newline on line %d, run %d, glyph %d", lineIdx, runIdx, glyphIdx)
						}
					}
				}
			}
			if t.Failed() {
				printLinePositioning(t, doc.lines, nil)
			}
		})
	}

}

// simpleGlyph returns a simple square glyph with the provided cluster
// value.
func simpleGlyph(cluster int) shaping.Glyph {
	return complexGlyph(cluster, 1, 1)
}

// ligatureGlyph returns a simple square glyph with the provided cluster
// value and number of runes.
func ligatureGlyph(cluster, runes int) shaping.Glyph {
	return complexGlyph(cluster, runes, 1)
}

// expansionGlyph returns a simple square glyph with the provided cluster
// value and number of glyphs.
func expansionGlyph(cluster, glyphs int) shaping.Glyph {
	return complexGlyph(cluster, 1, glyphs)
}

// complexGlyph returns a simple square glyph with the provided cluster
// value, number of associated runes, and number of glyphs in the cluster.
func complexGlyph(cluster, runes, glyphs int) shaping.Glyph {
	return shaping.Glyph{
		Width:        fixed.I(10),
		Height:       fixed.I(10),
		XAdvance:     fixed.I(10),
		YAdvance:     fixed.I(10),
		YBearing:     fixed.I(10),
		ClusterIndex: cluster,
		GlyphCount:   glyphs,
		RuneCount:    runes,
	}
}

// makeTestText creates a simple and complex(bidi) sample of shaped text at the given
// font size and wrapped to the given line width. The runeLimit, if nonzero,
// truncates the sample text to ensure shorter output for expensive tests.
func makeTestText(shaper *shaperImpl, primaryDir system.TextDirection, fontSize, lineWidth, runeLimit int) (simpleSample, complexSample []shaping.Line) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	if shaper == nil {
		shaper = testShaper(ltrFace, rtlFace)
	}

	ltrSource := "The quick brown fox jumps over the lazy dog."
	rtlSource := "الحب سماء لا تمط غير الأحلام"
	// bidiSource is crafted to contain multiple consecutive RTL runs (by
	// changing scripts within the RTL).
	bidiSource := "The quick سماء שלום لا fox تمط שלום غير the lazy dog."
	// bidi2Source is crafted to contain multiple consecutive LTR runs (by
	// changing scripts within the LTR).
	bidi2Source := "الحب سماء brown привет fox تمط jumps привет over غير الأحلام"

	locale := english
	simpleSource := ltrSource
	complexSource := bidiSource
	if primaryDir == system.RTL {
		simpleSource = rtlSource
		complexSource = bidi2Source
		locale = arabic
	}
	if runeLimit != 0 {
		simpleRunes := []rune(simpleSource)
		complexRunes := []rune(complexSource)
		if runeLimit < len(simpleRunes) {
			ltrSource = string(simpleRunes[:runeLimit])
		}
		if runeLimit < len(complexRunes) {
			rtlSource = string(complexRunes[:runeLimit])
		}
	}
	simpleText := shaper.shapeAndWrapText(shaper.orderer.sortedFacesForStyle(Font{}), Parameters{PxPerEm: fixed.I(fontSize)}, lineWidth, locale, []rune(simpleSource))
	complexText := shaper.shapeAndWrapText(shaper.orderer.sortedFacesForStyle(Font{}), Parameters{PxPerEm: fixed.I(fontSize)}, lineWidth, locale, []rune(complexSource))
	shaper = testShaper(rtlFace, ltrFace)
	return simpleText, complexText
}

func fixedAbs(a fixed.Int26_6) fixed.Int26_6 {
	if a < 0 {
		a = -a
	}
	return a
}

func TestToLine(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	shaper := testShaper(ltrFace, rtlFace)
	ltr, bidi := makeTestText(shaper, system.LTR, 16, 100, 0)
	rtl, bidi2 := makeTestText(shaper, system.RTL, 16, 100, 0)
	_, bidiWide := makeTestText(shaper, system.LTR, 16, 200, 0)
	_, bidi2Wide := makeTestText(shaper, system.RTL, 16, 200, 0)
	type testcase struct {
		name  string
		lines []shaping.Line
		// Dominant text direction.
		dir system.TextDirection
	}
	for _, tc := range []testcase{
		{
			name:  "ltr",
			lines: ltr,
			dir:   system.LTR,
		},
		{
			name:  "rtl",
			lines: rtl,
			dir:   system.RTL,
		},
		{
			name:  "bidi",
			lines: bidi,
			dir:   system.LTR,
		},
		{
			name:  "bidi2",
			lines: bidi2,
			dir:   system.RTL,
		},
		{
			name:  "bidi_wide",
			lines: bidiWide,
			dir:   system.LTR,
		},
		{
			name:  "bidi2_wide",
			lines: bidi2Wide,
			dir:   system.RTL,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// We expect:
			// - Line dimensions to be populated.
			// - Line direction to be populated.
			// - Runs to be ordered from lowest runes first.
			// - Runs to have widths matching the input.
			// - Runs to have the same total number of glyphs/runes as the input.
			runesSeen := Range{}
			shaper := testShaper(ltrFace, rtlFace)
			for i, input := range tc.lines {
				seenRun := make([]bool, len(input))
				inputLowestRuneOffset := math.MaxInt
				totalInputGlyphs := 0
				totalInputRunes := 0
				for _, run := range input {
					if run.Runes.Offset < inputLowestRuneOffset {
						inputLowestRuneOffset = run.Runes.Offset
					}
					totalInputGlyphs += len(run.Glyphs)
					totalInputRunes += run.Runes.Count
				}
				output := toLine(&shaper.orderer, input, tc.dir)
				if output.bounds.Min == (fixed.Point26_6{}) {
					t.Errorf("line %d: Bounds.Min not populated", i)
				}
				if output.bounds.Max == (fixed.Point26_6{}) {
					t.Errorf("line %d: Bounds.Max not populated", i)
				}
				if output.direction != tc.dir {
					t.Errorf("line %d: expected direction %v, got %v", i, tc.dir, output.direction)
				}
				totalRunWidth := fixed.I(0)
				totalLineGlyphs := 0
				totalLineRunes := 0
				for k, run := range output.runs {
					seenRun[run.VisualPosition] = true
					if output.visualOrder[run.VisualPosition] != k {
						t.Errorf("line %d, run %d: run.VisualPosition=%d, but line.VisualOrder[%d]=%d(should be %d)", i, k, run.VisualPosition, run.VisualPosition, output.visualOrder[run.VisualPosition], k)
					}
					if run.Runes.Offset != totalLineRunes {
						t.Errorf("line %d, run %d: expected Runes.Offset to be %d, got %d", i, k, totalLineRunes, run.Runes.Offset)
					}
					runGlyphCount := len(run.Glyphs)
					if inputGlyphs := len(input[k].Glyphs); runGlyphCount != inputGlyphs {
						t.Errorf("line %d, run %d: expected %d glyphs, found %d", i, k, inputGlyphs, runGlyphCount)
					}
					runRuneCount := 0
					currentCluster := -1
					for _, g := range run.Glyphs {
						if g.clusterIndex != currentCluster {
							runRuneCount += g.runeCount
							currentCluster = g.clusterIndex
						}
					}
					if run.Runes.Count != runRuneCount {
						t.Errorf("line %d, run %d: expected %d runes, counted %d", i, k, run.Runes.Count, runRuneCount)
					}
					runesSeen.Count += run.Runes.Count
					totalRunWidth += fixedAbs(run.Advance)
					totalLineGlyphs += len(run.Glyphs)
					totalLineRunes += run.Runes.Count
				}
				if output.runeCount != totalInputRunes {
					t.Errorf("line %d: input had %d runes, only counted %d", i, totalInputRunes, output.runeCount)
				}
				if totalLineGlyphs != totalInputGlyphs {
					t.Errorf("line %d: input had %d glyphs, only counted %d", i, totalInputRunes, totalLineGlyphs)
				}
				if totalRunWidth != output.width {
					t.Errorf("line %d: expected width %d, got %d", i, totalRunWidth, output.width)
				}
				for runIndex, seen := range seenRun {
					if !seen {
						t.Errorf("line %d, run %d missing from runs VisualPosition fields", i, runIndex)
					}
				}
			}
			lastLine := tc.lines[len(tc.lines)-1]
			maxRunes := 0
			for _, run := range lastLine {
				if run.Runes.Count+run.Runes.Offset > maxRunes {
					maxRunes = run.Runes.Count + run.Runes.Offset
				}
			}
			if runesSeen.Count != maxRunes {
				t.Errorf("input covered %d runes, output only covers %d", maxRunes, runesSeen.Count)
			}
		})
	}
}

func TestComputeVisualOrder(t *testing.T) {
	type testcase struct {
		name                string
		input               line
		expectedVisualOrder []int
	}
	for _, tc := range []testcase{
		{
			name: "ltr",
			input: line{
				direction: system.LTR,
				runs: []runLayout{
					{Direction: system.LTR},
					{Direction: system.LTR},
					{Direction: system.LTR},
				},
			},
			expectedVisualOrder: []int{0, 1, 2},
		},
		{
			name: "rtl",
			input: line{
				direction: system.RTL,
				runs: []runLayout{
					{Direction: system.RTL},
					{Direction: system.RTL},
					{Direction: system.RTL},
				},
			},
			expectedVisualOrder: []int{2, 1, 0},
		},
		{
			name: "bidi-ltr",
			input: line{
				direction: system.LTR,
				runs: []runLayout{
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.RTL},
					{Direction: system.RTL},
					{Direction: system.LTR},
				},
			},
			expectedVisualOrder: []int{0, 3, 2, 1, 4},
		},
		{
			name: "bidi-ltr-complex",
			input: line{
				direction: system.LTR,
				runs: []runLayout{
					{Direction: system.RTL},
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.RTL},
				},
			},
			expectedVisualOrder: []int{1, 0, 2, 4, 3, 5, 7, 6, 8, 10, 9},
		},
		{
			name: "bidi-rtl",
			input: line{
				direction: system.RTL,
				runs: []runLayout{
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.LTR},
					{Direction: system.LTR},
					{Direction: system.RTL},
				},
			},
			expectedVisualOrder: []int{4, 1, 2, 3, 0},
		},
		{
			name: "bidi-rtl-complex",
			input: line{
				direction: system.RTL,
				runs: []runLayout{
					{Direction: system.LTR},
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.LTR},
					{Direction: system.RTL},
					{Direction: system.LTR},
					{Direction: system.LTR},
				},
			},
			expectedVisualOrder: []int{9, 10, 8, 6, 7, 5, 3, 4, 2, 0, 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			computeVisualOrder(&tc.input)
			if !reflect.DeepEqual(tc.input.visualOrder, tc.expectedVisualOrder) {
				t.Errorf("expected visual order %v, got %v", tc.expectedVisualOrder, tc.input.visualOrder)
			}
			for i, visualIndex := range tc.input.visualOrder {
				if pos := tc.input.runs[visualIndex].VisualPosition; pos != i {
					t.Errorf("line.VisualOrder[%d]=%d, but line.Runs[%d].VisualPosition=%d", i, visualIndex, visualIndex, pos)
				}
			}
		})
	}
}

func FuzzLayout(f *testing.F) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	f.Add("د عرمثال dstي met لم aqل جدmوpمg lرe dرd  لو عل ميrةsdiduntut lab renنيتذدagلaaiua.ئPocttأior رادرsاي mيrbلmnonaيdتد ماةعcلخ.", true, uint8(10), uint16(200))

	shaper := testShaper(ltrFace, rtlFace)
	f.Fuzz(func(t *testing.T, txt string, rtl bool, fontSize uint8, width uint16) {
		locale := system.Locale{
			Direction: system.LTR,
		}
		if rtl {
			locale.Direction = system.RTL
		}
		if fontSize < 1 {
			fontSize = 1
		}
		lines := shaper.LayoutRunes(Parameters{PxPerEm: fixed.I(int(fontSize))}, 0, int(width), locale, []rune(txt))
		validateLines(t, lines.lines, len([]rune(txt)))
	})
}

func validateLines(t *testing.T, lines []line, expectedRuneCount int) {
	t.Helper()
	runesSeen := 0
	for i, line := range lines {
		if line.bounds.Min == (fixed.Point26_6{}) {
			t.Errorf("line %d: Bounds.Min not populated", i)
		}
		if line.bounds.Max == (fixed.Point26_6{}) {
			t.Errorf("line %d: Bounds.Max not populated", i)
		}
		totalRunWidth := fixed.I(0)
		totalLineGlyphs := 0
		lineRunesSeen := 0
		for k, run := range line.runs {
			if line.visualOrder[run.VisualPosition] != k {
				t.Errorf("line %d, run %d: run.VisualPosition=%d, but line.VisualOrder[%d]=%d(should be %d)", i, k, run.VisualPosition, run.VisualPosition, line.visualOrder[run.VisualPosition], k)
			}
			if run.Runes.Offset != lineRunesSeen {
				t.Errorf("line %d, run %d: expected Runes.Offset to be %d, got %d", i, k, lineRunesSeen, run.Runes.Offset)
			}
			runRuneCount := 0
			currentCluster := -1
			for _, g := range run.Glyphs {
				if g.clusterIndex != currentCluster {
					runRuneCount += g.runeCount
					currentCluster = g.clusterIndex
				}
			}
			if run.Runes.Count != runRuneCount {
				t.Errorf("line %d, run %d: expected %d runes, counted %d", i, k, run.Runes.Count, runRuneCount)
			}
			lineRunesSeen += run.Runes.Count
			totalRunWidth += fixedAbs(run.Advance)
			totalLineGlyphs += len(run.Glyphs)
		}
		if totalRunWidth != line.width {
			t.Errorf("line %d: expected width %d, got %d", i, line.width, totalRunWidth)
		}
		runesSeen += lineRunesSeen
	}
	if runesSeen != expectedRuneCount {
		t.Errorf("input covered %d runes, output only covers %d", expectedRuneCount, runesSeen)
	}
}

// TestTextAppend ensures that appending two texts together correctly updates the new lines'
// y offsets.
func TestTextAppend(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)

	shaper := testShaper(ltrFace, rtlFace)

	text1 := shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(14),
	}, 0, 200, english, "د عرمثال dstي met لم aqل جدmوpمg lرe dرd  لو عل ميrةsdiduntut lab renنيتذدagلaaiua.ئPocttأior رادرsاي mيrbلmnonaيdتد ماةعcلخ.")
	text2 := shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(14),
	}, 0, 200, english, "د عرمثال dstي met لم aqل جدmوpمg lرe dرd  لو عل ميrةsdiduntut lab renنيتذدagلaaiua.ئPocttأior رادرsاي mيrbلmnonaيdتد ماةعcلخ.")

	text1.append(text2)
	curY := math.MinInt
	for lineNum, line := range text1.lines {
		yOff := line.yOffset
		if yOff <= curY {
			t.Errorf("lines[%d] has y offset %d, <= to previous %d", lineNum, yOff, curY)
		}
		curY = yOff
	}
}

func TestClosestFontByWeight(t *testing.T) {
	const (
		testTF1 Typeface = "MockFace"
		testTF2 Typeface = "TestFace"
		testTF3 Typeface = "AnotherFace"
	)
	fonts := []Font{
		{Typeface: testTF1, Style: Regular, Weight: Normal},
		{Typeface: testTF1, Style: Regular, Weight: Light},
		{Typeface: testTF1, Style: Regular, Weight: Bold},
		{Typeface: testTF1, Style: Italic, Weight: Thin},
	}
	weightOnlyTests := []struct {
		Lookup   Weight
		Expected Weight
	}{
		// Test for existing weights.
		{Lookup: Normal, Expected: Normal},
		{Lookup: Light, Expected: Light},
		{Lookup: Bold, Expected: Bold},
		// Test for missing weights.
		{Lookup: Thin, Expected: Light},
		{Lookup: ExtraLight, Expected: Light},
		{Lookup: Medium, Expected: Normal},
		{Lookup: SemiBold, Expected: Bold},
		{Lookup: ExtraBlack, Expected: Bold},
	}
	for _, test := range weightOnlyTests {
		got, ok := closestFont(Font{Typeface: testTF1, Weight: test.Lookup}, fonts)
		if !ok {
			t.Errorf("expected closest font for %v to exist", test.Lookup)
		}
		if got.Weight != test.Expected {
			t.Errorf("got weight %v, expected %v", got.Weight, test.Expected)
		}
	}
	fonts = []Font{
		{Typeface: testTF1, Style: Regular, Weight: Light},
		{Typeface: testTF1, Style: Regular, Weight: Bold},
		{Typeface: testTF1, Style: Italic, Weight: Normal},
		{Typeface: testTF3, Style: Italic, Weight: Bold},
	}
	otherTests := []struct {
		Lookup         Font
		Expected       Font
		ExpectedToFail bool
	}{
		// Test for existing fonts.
		{
			Lookup:   Font{Typeface: testTF1, Weight: Light},
			Expected: Font{Typeface: testTF1, Style: Regular, Weight: Light},
		},
		{
			Lookup:   Font{Typeface: testTF1, Style: Italic, Weight: Normal},
			Expected: Font{Typeface: testTF1, Style: Italic, Weight: Normal},
		},
		// Test for missing fonts.
		{
			Lookup:   Font{Typeface: testTF1, Weight: Normal},
			Expected: Font{Typeface: testTF1, Style: Regular, Weight: Light},
		},
		{
			Lookup:   Font{Typeface: testTF3, Style: Italic, Weight: Normal},
			Expected: Font{Typeface: testTF3, Style: Italic, Weight: Bold},
		},
		{
			Lookup:   Font{Typeface: testTF1, Style: Italic, Weight: Thin},
			Expected: Font{Typeface: testTF1, Style: Italic, Weight: Normal},
		},
		{
			Lookup:   Font{Typeface: testTF1, Style: Italic, Weight: Bold},
			Expected: Font{Typeface: testTF1, Style: Italic, Weight: Normal},
		},
		{
			Lookup:         Font{Typeface: testTF2, Weight: Normal},
			ExpectedToFail: true,
		},
		{
			Lookup:         Font{Typeface: testTF2, Style: Italic, Weight: Normal},
			ExpectedToFail: true,
		},
	}
	for _, test := range otherTests {
		got, ok := closestFont(test.Lookup, fonts)
		if test.ExpectedToFail {
			if ok {
				t.Errorf("expected closest font for %v to not exist", test.Lookup)
			} else {
				continue
			}
		}
		if !ok {
			t.Errorf("expected closest font for %v to exist", test.Lookup)
		}
		if got != test.Expected {
			t.Errorf("got %v, expected %v", got, test.Expected)
		}
	}
}
