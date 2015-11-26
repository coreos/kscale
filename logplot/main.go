package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
)

func main() {
	fpath := flag.String("f", "data.txt", "data file path")
	dtype := flag.String("t", "density", "data type")
	flag.Parse()

	if *dtype != "density" {
		fmt.Fprintf(os.Stderr, "Unsupported data type: %s. Only support density.\n", dtype)
		os.Exit(1)
	}

	f, err := os.Open(*fpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %v\n", err)
		os.Exit(1)
	}

	plotDensity(parseDensity(f))
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
		var garbageInt int

		bytes, err := br.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			return results
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

		_, err = fmt.Sscanf(line, densityFormat, &r.created, &garbageInt, &r.running,
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

	// Save the plot to a PNG file.
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "density.svg"); err != nil {
		panic(err)
	}

	fmt.Println("successfully plotted density graph to density.svg")
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
