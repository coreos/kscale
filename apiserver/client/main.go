package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
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
	scaleNS = "scale-ns"
)

type rcJob struct {
	kubeClient *client.Client
}

func ExitError(msg string, args ...interface{}) {
	fmt.Println("exiting with error:")
	fmt.Printf(msg+"\n", args)
	os.Exit(1)
}

func main() {
	var apisrvAddr string
	var rcNum int
	var podNum int
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	flag.IntVar(&rcNum, "rc", 1000, "number of RC")
	flag.IntVar(&podNum, "pod", 100, "number of pods per RC")
	flag.Parse()

	c, err := createClient(apisrvAddr)
	if err != nil {
		ExitError("createClient failed: %v", err)
	}

	fmt.Printf("Creating %d rc, each of %d pods\n", rcNum, podNum)

	var wg sync.WaitGroup
	wg.Add(rcNum)
	for i := 0; i < rcNum; i++ {
		// TODO: clean up on failure
		go func(id int) {
			defer wg.Done()
			createRC(c, id, podNum)
			waitRCCreatePods(c, id, podNum)
		}(i)
	}
	wg.Wait()

	// introduce chaos
	fmt.Println("Success...")
}

func createRC(c *client.Client, id, podNum int) {
	rc := &api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: makeRCName(id),
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: int32(podNum),
			Selector: makeLabel(id),
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: makeLabel(id),
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "none",
							Image: "none",
						},
					},
				},
			},
		},
	}
	if _, err := c.ReplicationControllers(scaleNS).Create(rc); err != nil {
		ExitError("create RC failed: %v", err)
	}
	fmt.Printf("created rc: %s\n", makeRCName(id))
}

func waitRCCreatePods(c *client.Client, id, podNum int) {
	labelSelector := labels.SelectorFromSet(labels.Set(makeLabel(id)))

	stopCh := make(chan struct{})
	defer close(stopCh)

	total := 0
	finishChan := make(chan struct{})
	podStore, runner := controllerframework.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labelSelector
				return c.Pods(scaleNS).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector
				return c.Pods(scaleNS).Watch(options)
			},
		},
		&api.Pod{},
		0,
		controllerframework.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				total += 1
				if total == podNum {
					close(finishChan)
				}
			},
			DeleteFunc: func(obj interface{}) {
				total -= 1
			},
		},
	)

	go runner.Run(stopCh)
	for {
		select {
		case <-finishChan:
			fmt.Printf("rc (%s) created %d pods\n", makeRCName(id), podNum)
			return
		case <-time.After(1 * time.Minute):
			fmt.Printf("1 minute passed, rc (%s) has created %d pods\n", makeRCName(id), len(podStore.List()))
		}
	}
}

func updateRC(c *client.Client) {
	rc := &api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: "wan-rc",
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: 0,
			Selector: map[string]string{
				"name": "wan-label",
			},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{"name": "wan-label"},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "none",
							Image: "none",
						},
					},
				},
			},
		},
	}
	if _, err := c.ReplicationControllers("wan-ns").Update(rc); err != nil {
		ExitError("update RC failed: %v", err)
	}
}

func deleteRC(c *client.Client) {
	if err := c.ReplicationControllers("wan-ns").Delete("wan-rc"); err != nil {
		ExitError("create RC failed: %v", err)
	}
}

func createClient(addr string) (*client.Client, error) {
	cfg := &restclient.Config{
		Host:  fmt.Sprintf("http://%s", addr),
		QPS:   1000,
		Burst: 1000,
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func makeRCName(id int) string {
	return fmt.Sprintf("scale-rc-%d", id)
}

func makeLabel(id int) map[string]string {
	return map[string]string{"name": fmt.Sprintf("scale-label-%d", id)}
}
