package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
)

// total number of pods to be scheduled and run
var totalPods int

func main() {
	fpath := flag.String("f", "data.txt", "data file path")
	dtype := flag.String("t", "density", "data type")
	flag.Parse()

	if *dtype != "density" {
		fmt.Fprintf(os.Stderr, "Unsupported data type: %s. Only support density.\n", *dtype)
		os.Exit(1)
	}

	f, err := os.Open(*fpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %v\n", err)
		os.Exit(1)
	}

	rs := parseDensity(f)
	plotDensity(rs)
	plotCreatingRateVsPods(rs)
	plotRunningRateVsPods(rs)
	recordAvgRunningRate(rs)
}

type densityResult struct {
	// x axis
	seconds int

	// y axis
	created int
	running int
	pending int
	waiting int
}

// wanted format:
// Nov 25 23:05:18.250: INFO: densityN-X Pods: 12000 out of 12000 created, 1012 running,
// 23 pending, 10965 waiting, 0 inactive, 0 terminating, 0 unknown, 0 runningButNotReady
func parseDensity(r io.Reader) (results []densityResult) {
	densityFormat := "Pods: %d out of %d created, %d running, %d pending, %d waiting, %s"
	//10 seconds interval
	interval := 10
	seconds := interval

	br := bufio.NewReader(r)
	for {
		var r densityResult
		var garbageString string

		bytes, err := br.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}

		line := strings.TrimSpace(string(bytes[:len(bytes)-1]))

		if !strings.HasSuffix(line, "runningButNotReady") {
			continue
		}

		pi := strings.Index(line, "Pods")
		if pi == -1 {
			fmt.Fprintln(os.Stderr, "Bad density format: cannot find Pods")
			os.Exit(1)
		}

		line = line[pi:]

		_, err = fmt.Sscanf(line, densityFormat, &r.created, &totalPods, &r.running,
			&r.pending, &r.waiting, &garbageString)

		if err != nil {
			fmt.Fprintln(os.Stderr, "Bad density format:", err)
			os.Exit(1)
		}

		r.seconds = seconds
		seconds += interval

		results = append(results, r)
	}
	return results
}

func plotDensity(results []densityResult) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = "Density"
	p.X.Label.Text = "Seconds"
	p.Y.Label.Text = "Number of Pods"

	err = plotutil.AddLinePoints(p,
		"Created", getCreatedPoints(results),
		"Running", getRunningPoints(results),
		"Pending", getPendingPoints(results),
		"Waiting", getWaitingPoints(results))
	if err != nil {
		panic(err)
	}

	// Save the plot to a SVG file.
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "density-all.svg"); err != nil {
		panic(err)
	}

	fmt.Println("successfully plotted density graph to density-all.svg")
}

func plotCreatingRateVsPods(rs []densityResult) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = "CreatingRate"
	p.X.Label.Text = "Number of Pods"
	p.Y.Label.Text = "Rate"

	err = plotutil.AddLinePoints(p, "CreatingRate", getCreatingRatePoints(rs))
	if err != nil {
		panic(err)
	}

	// Save the plot to a SVG file.
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "density-creating-rate.svg"); err != nil {
		panic(err)
	}

	fmt.Println("successfully plotted density graph to density-creating-rate.svg")
}

func plotRunningRateVsPods(rs []densityResult) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = "RunningRate"
	p.X.Label.Text = "Number of Pods"
	p.Y.Label.Text = "Rate"

	err = plotutil.AddLinePoints(p, "RunningRate", getRunningRatePoints(rs))
	if err != nil {
		panic(err)
	}

	// Save the plot to a SVG file.
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "density-running-rate.svg"); err != nil {
		panic(err)
	}

	fmt.Println("successfully plotted density graph to density-running-rate.svg")
}

func recordAvgRunningRate(rs []densityResult) {
	f, err := os.Create("avg-running-rate.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot create 'avg-running-rate.txt' file")
		os.Exit(1)
	}

	defer f.Close()

	r := getAvgRunningRate(rs)
	s := strconv.FormatFloat(r, 'f', 6, 64)
	_, err = f.WriteString(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write to 'avg-running-rate.txt'")
		os.Exit(1)
	}
	fmt.Println("successfully write average running rate to avg-running-rate.txt")
}

func getCreatedPoints(rs []densityResult) plotter.XYs {
	pts := make(plotter.XYs, len(rs))

	for i := range rs {
		pts[i].X = float64(rs[i].seconds)
		pts[i].Y = float64(rs[i].created)
	}
	return pts
}

func getRunningPoints(rs []densityResult) plotter.XYs {
	pts := make(plotter.XYs, len(rs))

	for i := range rs {
		pts[i].X = float64(rs[i].seconds)
		pts[i].Y = float64(rs[i].running)
	}
	return pts
}

func getPendingPoints(rs []densityResult) plotter.XYs {
	pts := make(plotter.XYs, len(rs))

	for i := range rs {
		pts[i].X = float64(rs[i].seconds)
		pts[i].Y = float64(rs[i].pending)
	}
	return pts
}

func getWaitingPoints(rs []densityResult) plotter.XYs {
	pts := make(plotter.XYs, len(rs))

	for i := range rs {
		pts[i].X = float64(rs[i].seconds)
		pts[i].Y = float64(rs[i].waiting)
	}
	return pts
}

func getCreatingRatePoints(rs []densityResult) plotter.XYs {
	pts := make(plotter.XYs, len(rs))
	interval := 10

	for i := range rs {
		if i == 0 {
			continue
		}
		pts[i].X = float64(rs[i].created)
		pts[i].Y = float64(rs[i].created-rs[i-1].created) / float64(interval)
	}
	return pts
}

func getRunningRatePoints(rs []densityResult) plotter.XYs {
	pts := make(plotter.XYs, len(rs))
	interval := 10

	for i := range rs {
		if i == 0 {
			continue
		}
		pts[i].X = float64(rs[i].running)
		pts[i].Y = float64(rs[i].running-rs[i-1].running) / float64(interval)
	}
	return pts
}

func getAvgRunningRate(rs []densityResult) float64 {
	interval := 10
	for i := range rs {
		if rs[i].running >= totalPods {
			return float64(rs[i].running) / float64((i+1)*interval)
		}
	}
	n := len(rs)
	return float64(rs[n-1].running) / float64(n*interval)
}
