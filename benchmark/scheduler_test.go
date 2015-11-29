package benchmark

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/apiserver"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/master"
	"k8s.io/kubernetes/plugin/pkg/admission/admit"
	"k8s.io/kubernetes/plugin/pkg/scheduler"
	_ "k8s.io/kubernetes/plugin/pkg/scheduler/algorithmprovider"
	"k8s.io/kubernetes/plugin/pkg/scheduler/factory"
	"k8s.io/kubernetes/test/integration/framework"
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

func benchmarkScheduling(n, p int, b *testing.B) {
	etcdStorage, err := framework.NewEtcdStorage()
	if err != nil {
		b.Fatalf("Couldn't create etcd storage: %v", err)
	}
	expEtcdStorage, err := framework.NewExtensionsEtcdStorage(nil)
	if err != nil {
		b.Fatalf("Unexpected error: %v", err)
	}

	storageDestinations := master.NewStorageDestinations()
	storageDestinations.AddAPIGroup("", etcdStorage)
	storageDestinations.AddAPIGroup("extensions", expEtcdStorage)

	storageVersions := make(map[string]string)
	storageVersions[""] = testapi.Default.Version()
	storageVersions["extensions"] = testapi.Extensions.GroupAndVersion()

	framework.DeleteAllEtcdKeys()

	var m *master.Master
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m.Handler.ServeHTTP(w, req)
	}))
	defer s.Close()

	m = master.New(&master.Config{
		StorageDestinations:   storageDestinations,
		KubeletClient:         client.FakeKubeletClient{},
		EnableCoreControllers: true,
		EnableLogsSupport:     false,
		EnableUISupport:       false,
		EnableIndex:           true,
		APIPrefix:             "/api",
		Authorizer:            apiserver.NewAlwaysAllowAuthorizer(),
		AdmissionControl:      admit.NewAlwaysAdmit(),
		StorageVersions:       storageVersions,
	})

	c := client.NewOrDie(&client.Config{
		Host:         s.URL,
		GroupVersion: testapi.Default.GroupVersion(),
		QPS:          5000.0,
		Burst:        5000,
	})

	schedulerConfigFactory := factory.NewConfigFactory(c, nil)
	schedulerConfig, err := schedulerConfigFactory.Create()
	if err != nil {
		b.Fatalf("Couldn't create scheduler config: %v", err)
	}

	eventBroadcaster := record.NewBroadcaster()
	schedulerConfig.Recorder = eventBroadcaster.NewRecorder(api.EventSource{Component: "scheduler"})
	eventBroadcaster.StartRecordingToSink(c.Events(""))
	scheduler.New(schedulerConfig).Run()

	defer close(schedulerConfig.StopEverything)

	// prepare N nodes with P pods.
	makeNNodes(c, n)
	numPods := p
	makeNPods(c, numPods)
	for {
		objs := schedulerConfigFactory.ScheduledPodLister.Store.List()
		if len(objs) >= numPods {
			glog.Infof("%v pods scheduled.\n", len(objs))
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// start benchmark
	b.ResetTimer()
	makeNPods(c, b.N)
	for {
		objs := schedulerConfigFactory.ScheduledPodLister.Store.List()
		if len(objs) >= numPods+b.N {
			glog.Infof("%v pods scheduled.\n", len(objs))
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	b.StopTimer()
}

func makeNNodes(c client.Interface, N int) {
	baseNode := &api.Node{
		ObjectMeta: api.ObjectMeta{
			GenerateName: "scheduler-test-node-",
		},
		Spec: api.NodeSpec{
			ExternalID: "foobar",
		},
		Status: api.NodeStatus{
			Capacity: api.ResourceList{
				api.ResourcePods:   *resource.NewQuantity(32, resource.DecimalSI),
				api.ResourceCPU:    resource.MustParse("4"),
				api.ResourceMemory: resource.MustParse("32Gi"),
			},
			Phase: api.NodeRunning,
			Conditions: []api.NodeCondition{
				{Type: api.NodeReady, Status: api.ConditionTrue},
			},
		},
	}
	for i := 0; i < N; i++ {
		if _, err := c.Nodes().Create(baseNode); err != nil {
			panic("error creating node: " + err.Error())
		}
	}
}

func makeNPods(c client.Interface, N int) {
	basePod := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			GenerateName: "scheduler-test-pod-",
		},
		Spec: api.PodSpec{
			Containers: []api.Container{{
				Name:  "pause",
				Image: "gcr.io/google_containers/pause:1.0",
				Resources: api.ResourceRequirements{
					Limits: api.ResourceList{
						api.ResourceCPU:    resource.MustParse("100m"),
						api.ResourceMemory: resource.MustParse("500Mi"),
					},
					Requests: api.ResourceList{
						api.ResourceCPU:    resource.MustParse("100m"),
						api.ResourceMemory: resource.MustParse("500Mi"),
					},
				},
			}},
		},
	}
	wg := sync.WaitGroup{}
	threads := 30
	wg.Add(threads)
	remaining := make(chan int, N)
	go func() {
		for i := 0; i < N; i++ {
			remaining <- i
		}
		close(remaining)
	}()
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			for {
				_, ok := <-remaining
				if !ok {
					return
				}
				for {
					_, err := c.Pods("default").Create(basePod)
					if err == nil {
						break
					}
				}
			}
		}()
	}
	wg.Wait()
}
