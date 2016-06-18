package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
)

func main() {
	f, err := os.Open("pod.txt")
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(f)
	pod := &api.Pod{}
	if err := dec.Decode(pod); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", pod)

	// simulate store of 100k pods
	rep := make(map[string]*api.Pod)
	for i := 0; i < 100000; i++ {
		key, err := cache.MetaNamespaceKeyFunc(pod)
		if err != nil {
			panic(err)
		}
		key = fmt.Sprintf("%s-%d", key, i)
		newPod := *pod
		rep[key] = &newPod
	}

	notifier := make(chan os.Signal, 1)
	signal.Notify(notifier, os.Interrupt, os.Kill)
	fmt.Println("waiting for signal")
	sig := <-notifier
	fmt.Printf("sig: %v\n", sig)
	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	fmt.Printf("alloc: %d, sys: %d, idle: %d, inuse: %d\n", st.HeapAlloc, st.HeapSys, st.HeapIdle, st.HeapInuse)
}
