package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	controllerframework "k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	scaleNSPrefix = "scale-ns"
)

var podMarkerSize int
var apisrvAddr string
var nsNum int
var rcNum int
var podNum int
var freshCluster bool
var chaosEnabled bool

var garbage []byte

func init() {
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	flag.IntVar(&nsNum, "ns", 100, "number of namespaces")
	flag.IntVar(&rcNum, "rc", 10, "number of RC per namespace")
	flag.IntVar(&podNum, "pod", 100, "number of pods per RC")
	flag.IntVar(&podMarkerSize, "pod-size", 0, "pod marker size in kb")
	flag.BoolVar(&freshCluster, "fresh", true, "fresh cluster? We will create pods if so.")
	flag.BoolVar(&chaosEnabled, "chaos", true, "running chaos testing only")
	flag.Parse()

	garbage = make([]byte, podMarkerSize*1024)
	for i := 0; i < podMarkerSize*1024; i++ {
		garbage[i] = 0x30
	}
}

type rcJob struct {
	kubeClient *client.Client
}

func ExitError(msg string, args ...interface{}) {
	fmt.Println("exiting with error:")
	fmt.Printf(msg+"\n", args...)
	debug.PrintStack()
	os.Exit(1)
}

func main() {
	c, err := createClient(apisrvAddr)
	if err != nil {
		ExitError("createClient failed: %v", err)
	}

	fmt.Printf("Creating %d ns X %d rc X %d pods = %d\n", nsNum, rcNum, podNum, nsNum*rcNum*podNum)

	if freshCluster {
		createPods(c, nsNum, rcNum, podNum)
		fmt.Println("creation phase is done...")
		time.Sleep(1 * time.Second)
	}

	// NOTE: due to introducing RC per namespace, this is not supported temporarily.
	//
	// if chaosEnabled {
	// 	var wg sync.WaitGroup
	// 	wg.Add(rcNum)
	// 	for i := 0; i < rcNum; i++ {
	// 		// TODO: clean up on failure
	// 		go func(id int) {
	// 			defer wg.Done()
	// 			deletePodsRandomly(c, id, podNum)
	// 			waitRCCreatePods(c, id, podNum)
	// 		}(i)
	// 	}
	// 	wg.Wait()
	// }

	fmt.Println("Success...")
}

func createPods(c *client.Client, nsNum, rcNum, podNum int) {
	for i := 0; i < nsNum; i++ {
		for j := 0; j < rcNum; j++ {
			go createRC(c, i, j, podNum)
		}
	}
	waitRCCreatePods(c, nsNum*rcNum*podNum)
}

func createRC(c *client.Client, nsID, rcID, podNum int) {
	var args []string
	if podMarkerSize != 0 {
		args = []string{string(garbage)}
	}
	rc := &api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: makeRCName(rcID),
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: int32(podNum),
			Selector: makeLabel(nsID, rcID),
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: makeLabel(nsID, rcID),
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "none",
							Image: "none",
							Args:  args,
						},
					},
				},
			},
		},
	}
	if _, err := c.ReplicationControllers(makeNS(nsID)).Create(rc); err != nil {
		ExitError("create rc (%s/%s), failed: %v", makeNS(nsID), makeRCName(rcID), err)
	}
	fmt.Printf("created rc (%s'%s)\n", makeNS(nsID), makeRCName(rcID))
}

func waitRCCreatePods(c *client.Client, podNum int) {
	// Currently this is inefficient. It will watch all pods under the namespace.
	informer := createPodInformer(c)

	doneCh := make(chan struct{})
	total := 0
	informer.AddEventHandler(controllerframework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			total += 1
			if total == podNum {
				close(doneCh)
			}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	go informer.Run(stopCh)

	store := informer.GetStore()
	start := time.Now()
	for {
		select {
		case <-doneCh:
			fmt.Printf("created %d pods\n", podNum)
			return
		case <-time.After(1 * time.Minute):
			fmt.Printf("After %v, created %d pods\n", time.Since(start), len(store.List()))
		}
	}
}

func deletePodsRandomly(c *client.Client, nsID, rcID, podNum int) {
	podList := listPods(c, nsID, rcID)
	for i, pod := range podList.Items {
		if i%2 != 0 {
			continue
		}
		if err := c.Pods(makeNS(nsID)).Delete(pod.Name, api.NewDeleteOptions(0)); err != nil {
			ExitError("delete pod (%s/%s) failed: %v", makeNS(nsID), pod.Name, err)
		}
		fmt.Printf("rc (%s/%s) deleted pod %s\n", makeNS(nsID), makeRCName(rcID), pod.Name)
	}
}

func listPods(c *client.Client, nsID, rcID int) *api.PodList {
	podList, err := c.Pods(makeNS(nsID)).List(api.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(makeLabel(nsID, rcID))),
	})
	if err != nil {
		ExitError("list pods failed: %v", err)
	}
	return podList
}

func createPodInformer(c *client.Client) controllerframework.SharedInformer {
	informer := controllerframework.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return c.Pods(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return c.Pods(api.NamespaceAll).Watch(options)
			},
		},
		&api.Pod{},
		0,
	)
	return informer
}

func createClient(addr string) (*client.Client, error) {
	cfg := &restclient.Config{
		Host:  fmt.Sprintf("http://%s", addr),
		QPS:   100,
		Burst: 100,
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func makeNS(id int) string {
	return fmt.Sprintf("%s-%d", scaleNSPrefix, id)
}

func makeRCName(id int) string {
	return fmt.Sprintf("scale-rc-%d", id)
}

func makeLabel(nsID, rcID int) map[string]string {
	return map[string]string{"name": fmt.Sprintf("scale-label-%d-%d", nsID, rcID)}
}
