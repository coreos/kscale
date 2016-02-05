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

type result struct {
	// x axis
	time int

	// y axis
	rate  int
	total int
}

func main() {
	nfpath := flag.String("nf", "bench-new.txt", "data file path")
	ofpath := flag.String("of", "bench-old.txt", "data file path")
	ftype := flag.String("t", "total", "rate, total")
	flag.Parse()

	nf, err := os.Open(*nfpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %s, err: %v\n", *nfpath, err)
		os.Exit(1)
	}
	of, err := os.Open(*ofpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %s, err: %v\n", *ofpath, err)
		os.Exit(1)
	}

	rs1, rs2 := parseDensity(nf, of)
	plotDensity(rs1, rs2, *ftype)
}

func parseDensity(nr, or io.Reader) (res1, res2 []result) {
	densityFormat := "%ds\trate: %d\ttotal: %d"

	nbr := bufio.NewReader(nr)
	obr := bufio.NewReader(or)
	for {
		var r1 result

		bytes, err := nbr.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}

		line := strings.TrimSpace(string(bytes[:len(bytes)-1]))

		_, err = fmt.Sscanf(line, densityFormat, &r1.time, &r1.rate, &r1.total)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Bad density format:", err)
			os.Exit(1)
		}

		res1 = append(res1, r1)
	}

	for {
		var r2 result

		bytes, err := obr.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}

		line := strings.TrimSpace(string(bytes[:len(bytes)-1]))

		_, err = fmt.Sscanf(line, densityFormat, &r2.time, &r2.rate, &r2.total)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Bad density format:", err)
			os.Exit(1)
		}

		res2 = append(res2, r2)
	}
	return
}

func plotDensity(rs1, rs2 []result, ftype string) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = "Scheduler Benchmark"
	p.X.Label.Text = "Seconds"

	var filename string
	switch ftype {
	case "total":
		p.Y.Label.Text = "Number of Pods"
		err = plotutil.AddLinePoints(p,
			"Total-new", getTotal(rs1),
			"Total-old", getTotal(rs2))
		filename = "schedule-total.png"
	case "rate":
		p.Y.Label.Text = "Rate of Scheduling"
		err = plotutil.AddLinePoints(p,
			"Rate-new", getRate(rs1),
			"Rate-old", getRate(rs2))
		filename = "schedule-rate.png"
	}
	if err != nil {
		panic(err)
	}

	if err := p.Save(10*vg.Inch, 10*vg.Inch, filename); err != nil {
		panic(err)
	}

	fmt.Println("successfully plotted density graph to", filename)
}

func getRate(rs []result) plotter.XYs {
	pts := make(plotter.XYs, len(rs))

	for i := range rs {
		pts[i].X = float64(rs[i].time)
		pts[i].Y = float64(rs[i].rate)
	}
	return pts
}

func getTotal(rs []result) plotter.XYs {
	pts := make(plotter.XYs, len(rs))

	for i := range rs {
		pts[i].X = float64(rs[i].time)
		pts[i].Y = float64(rs[i].total)
	}
	return pts
}
