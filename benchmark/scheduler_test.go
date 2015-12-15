package benchmark

import (
	"fmt"
	"testing"
	"time"
)

// TestScheduling1000Nodes tests the scheduler to schedule
// 10K pods over 1000 nodes.
// The test might take up to 1 hour.
// The test prints out the number of scheduled pods every 1 second.

// TODO: bench and test should share the same setup code.
func TestScheduling1000Nodes10KPods(t *testing.T) {
	schedulerConfigFactory, destroyFunc := mustSetupScheduler()
	defer destroyFunc()
	c := schedulerConfigFactory.Client

	numPods := 10000
	numNodes := 1000
	makeNodes(c, numNodes)
	makePods(c, numPods)

	prev := 0
	start := time.Now()
	for {
		scheduled := schedulerConfigFactory.ScheduledPodLister.Store.List()
		fmt.Printf("%ds rate: %d total: %d\n", time.Since(start)/time.Second, len(scheduled)-prev, len(scheduled))
		if len(scheduled) >= numPods {
			return
		}
		prev = len(scheduled)
		time.Sleep(1 * time.Second)
	}
}
