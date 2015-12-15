package benchmark

import (
	"testing"
	"time"
)

// BenchmarkScheduling100Nodes0Pods benchmarks the scheduling rate
// when the cluster has 100 nodes and 0 scheduled pods
func BenchmarkScheduling100Nodes0Pods(b *testing.B) {
	benchmarkScheduling(100, 0, b)
}

// BenchmarkScheduling100Nodes1000Pods benchmarks the scheduling rate
// when the cluster has 100 nodes and 1000 scheduled pods
func BenchmarkScheduling100Nodes1000Pods(b *testing.B) {
	benchmarkScheduling(100, 1000, b)
}

// BenchmarkScheduling1000Nodes0Pods benchmarks the scheduling rate
// when the cluster has 1000 nodes and 0 scheduled pods
func BenchmarkScheduling1000Nodes0Pods(b *testing.B) {
	benchmarkScheduling(1000, 0, b)
}

// BenchmarkScheduling1000Nodes10000Pods benchmarks the scheduling rate
// when the cluster has 1000 nodes and 1000 scheduled pods
func BenchmarkScheduling1000Nodes1000Pods(b *testing.B) {
	benchmarkScheduling(1000, 1000, b)
}

func benchmarkScheduling(numNodes, numPods int, b *testing.B) {
	schedulerConfigFactory, finalFunc := mustSetupScheduler()
	defer finalFunc()
	c := schedulerConfigFactory.Client

	makeNodes(c, numNodes)
	makePods(c, numPods)
	for {
		scheduled := schedulerConfigFactory.ScheduledPodLister.Store.List()
		if len(scheduled) >= numPods {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	// start benchmark
	b.ResetTimer()
	makePods(c, b.N)
	for {
		scheduled := schedulerConfigFactory.ScheduledPodLister.Store.List()
		if len(scheduled) >= numPods+b.N {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	b.StopTimer()
}
