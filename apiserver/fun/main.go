package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"k8s.io/kubernetes/pkg/api"
)

var store map[int]*api.Pod

func main() {
	var filename string
	flag.StringVar(&filename, "f", "pod.txt", "pod json data")
	flag.Parse()

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()

	store = make(map[int]*api.Pod)
	for i := 0; i < 100000; i++ {
		pod := &api.Pod{}
		if err := json.Unmarshal(data, pod); err != nil {
			panic(err)
		}
		store[i] = pod
	}

	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	fmt.Printf("alloc: %d, sys: %d, idle: %d, inuse: %d\n", st.HeapAlloc, st.HeapSys, st.HeapIdle, st.HeapInuse)
}
