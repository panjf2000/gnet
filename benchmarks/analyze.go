package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

var category string
var kind string
var connections, commands, pipeline, seconds int
var rate float64
var values []float64
var names []string

func main() {
	analyze()
	autoplot()
}

func autoplot() {
	if category == "" {
		return
	}
	var title = category
	path := strings.Replace("out/"+category+".png", " ", "_", -1)

	plotit(
		path,
		title,
		values,
		names,
	)

}

func analyze() {
	lines := readlines("out/http.txt", "out/echo.txt", "out/redis1.txt", "out/redis8.txt", "out/redis16.txt")
	var err error
	for _, line := range lines {
		rlines := strings.Split(line, "\r")
		line = strings.TrimSpace(rlines[len(rlines)-1])
		if strings.HasPrefix(line, "--- ") {
			if strings.HasSuffix(line, " START ---") {
				autoplot()
				category = strings.ToLower(strings.Replace(strings.Replace(line, "--- ", "", -1), " START ---", "", -1))
				category = strings.Replace(category, "bench ", "", -1)
				values = nil
				names = nil
			} else {
				kind = strings.ToLower(strings.Replace(strings.Replace(line, "--- ", "", -1), " ---", "", -1))
			}
			connections, commands, pipeline, seconds = 0, 0, 0, 0
		} else if strings.HasPrefix(line, "*** ") {
			details := strings.Split(strings.ToLower(strings.Replace(line, "*** ", "", -1)), ", ")
			for _, item := range details {
				if strings.HasSuffix(item, " connections") {
					connections, err = strconv.Atoi(strings.Split(item, " ")[0])
					must(err)
				} else if strings.HasSuffix(item, " commands") {
					commands, err = strconv.Atoi(strings.Split(item, " ")[0])
					must(err)
				} else if strings.HasSuffix(item, " commands pipeline") {
					pipeline, err = strconv.Atoi(strings.Split(item, " ")[0])
					must(err)

				} else if strings.HasSuffix(item, " seconds") {
					seconds, err = strconv.Atoi(strings.Split(item, " ")[0])
					must(err)
				}
			}
		} else {
			switch {
			case category == "echo":
				if strings.HasPrefix(line, "Packet rate estimate: ") {
					rate, err = strconv.ParseFloat(strings.Split(strings.Split(line, ": ")[1], "↓,")[0], 64)
					must(err)
					output()
				}
			case category == "http":
				if strings.HasPrefix(line, "Reqs/sec ") {
					rate, err = strconv.ParseFloat(
						strings.Split(strings.TrimSpace(strings.Split(line, "Reqs/sec ")[1]), " ")[0], 64)
					must(err)
					output()
				}
			case strings.HasPrefix(category, "redis"):
				if strings.HasPrefix(line, "PING_INLINE: ") {
					rate, err = strconv.ParseFloat(strings.Split(strings.Split(line, ": ")[1], " ")[0], 64)
					must(err)
					output()
				}
			}
		}
	}
}

func output() {
	name := kind
	names = append(names, name)
	values = append(values, rate)
	//csv += fmt.Sprintf("%s,%s,%d,%d,%d,%d,%f\n", category, kind, connections, commands, pipeline, seconds, rate)
}

func readlines(paths ...string) (lines []string) {
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		must(err)
		lines = append(lines, strings.Split(string(data), "\n")...)
	}
	return
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func plotit(path, title string, values []float64, names []string) {
	plot.DefaultFont = "Helvetica"
	var groups []plotter.Values
	for _, value := range values {
		groups = append(groups, plotter.Values{value})
	}
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = title
	p.Y.Tick.Marker = commaTicks{}
	p.Y.Label.Text = "Req/s"
	bw := 25.0
	w := vg.Points(bw)
	var bars []plot.Plotter
	var barsp []*plotter.BarChart
	for i := 0; i < len(values); i++ {
		bar, err := plotter.NewBarChart(groups[i], w)
		if err != nil {
			panic(err)
		}
		bar.LineStyle.Width = vg.Length(0)
		bar.Color = plotutil.Color(i)
		bar.Offset = vg.Length(
			(float64(w) * float64(i)) -
				(float64(w)*float64(len(values)))/2)
		bars = append(bars, bar)
		barsp = append(barsp, bar)
	}
	p.Add(bars...)
	for i, name := range names {
		p.Legend.Add(fmt.Sprintf("%s (%.0f req/s)", name, values[i]), barsp[i])
	}

	p.Legend.Top = true
	p.NominalX("")

	if err := p.Save(7*vg.Inch, 3*vg.Inch, path); err != nil {
		panic(err)
	}
}

// PreciseTicks is suitable for the Tick.Marker field of an Axis, it returns a
// set of tick marks with labels that have been rounded less agressively than
// what DefaultTicks provides.
type PreciseTicks struct{}

// Ticks returns Ticks in a specified range
func (PreciseTicks) Ticks(min, max float64) []plot.Tick {
	const suggestedTicks = 3

	if max <= min {
		panic("illegal range")
	}

	tens := math.Pow10(int(math.Floor(math.Log10(max - min))))
	n := (max - min) / tens
	for n < suggestedTicks-1 {
		tens /= 10
		n = (max - min) / tens
	}

	majorMult := int(n / (suggestedTicks - 1))
	switch majorMult {
	case 7:
		majorMult = 6
	case 9:
		majorMult = 8
	}
	majorDelta := float64(majorMult) * tens
	val := math.Floor(min/majorDelta) * majorDelta
	// Makes a list of non-truncated y-values.
	var labels []float64
	for val <= max {
		if val >= min {
			labels = append(labels, val)
		}
		val += majorDelta
	}
	prec := int(math.Ceil(math.Log10(val)) - math.Floor(math.Log10(majorDelta)))
	// Makes a list of big ticks.
	var ticks []plot.Tick
	for _, v := range labels {
		vRounded := round(v, prec)
		ticks = append(ticks, plot.Tick{Value: vRounded, Label: strconv.FormatFloat(vRounded, 'f', -1, 64)})
	}
	minorDelta := majorDelta / 2
	switch majorMult {
	case 3, 6:
		minorDelta = majorDelta / 3
	case 5:
		minorDelta = majorDelta / 5
	}

	val = math.Floor(min/minorDelta) * minorDelta
	for val <= max {
		found := false
		for _, t := range ticks {
			if t.Value == val {
				found = true
			}
		}
		if val >= min && val <= max && !found {
			ticks = append(ticks, plot.Tick{Value: val})
		}
		val += minorDelta
	}
	return ticks
}

type commaTicks struct{}

// Ticks computes the default tick marks, but inserts commas
// into the labels for the major tick marks.
func (commaTicks) Ticks(min, max float64) []plot.Tick {
	tks := PreciseTicks{}.Ticks(min, max)
	for i, t := range tks {
		if t.Label == "" { // Skip minor ticks, they are fine.
			continue
		}
		tks[i].Label = addCommas(t.Label)
	}
	return tks
}

// AddCommas adds commas after every 3 characters from right to left.
// NOTE: This function is a quick hack, it doesn't work with decimal
// points, and may have a bunch of other problems.
func addCommas(s string) string {
	rev := ""
	n := 0
	for i := len(s) - 1; i >= 0; i-- {
		rev += string(s[i])
		n++
		if n%3 == 0 {
			rev += ","
		}
	}
	s = ""
	for i := len(rev) - 1; i >= 0; i-- {
		s += string(rev[i])
	}
	if strings.HasPrefix(s, ",") {
		s = s[1:]
	}
	return s
}

// round returns the half away from zero rounded value of x with a prec precision.
//
// Special cases are:
// 	round(±0) = +0
// 	round(±Inf) = ±Inf
// 	round(NaN) = NaN
func round(x float64, prec int) float64 {
	if x == 0 {
		// Make sure zero is returned
		// without the negative bit set.
		return 0
	}
	// Fast path for positive precision on integers.
	if prec >= 0 && x == math.Trunc(x) {
		return x
	}
	pow := math.Pow10(prec)
	intermed := x * pow
	if math.IsInf(intermed, 0) {
		return x
	}
	if x < 0 {
		x = math.Ceil(intermed - 0.5)
	} else {
		x = math.Floor(intermed + 0.5)
	}

	if x == 0 {
		return 0
	}

	return x / pow
}
