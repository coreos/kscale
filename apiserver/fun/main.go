package main

import (
	"encoding/json"
	"fmt"
	"os"

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
		rep[key] = pod
	}
	for {
	}
}
