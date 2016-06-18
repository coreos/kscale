package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"runtime"

	"k8s.io/kubernetes/pkg/api"
)

var store map[int]*api.Pod

// let's get the plain size of a Pod in memory
// assume each field takes about 16 bytes additional space
// unversioned.TypeMeta = 2 -> 32 bytes
// ObjectMeta = 16 -> 256 bytes
// PodSpec = 14 -> 224 bytes
// PodStatus = 9 -> 144 bytes
// Pod -> 4 -> 64 bytes
// total = 45 -> 720 bytes

// calculation time!
// 100k empty pods!
// 72105984 bytes = 721 bytes -> wow accurate

// OK. Now let's fill in some fields to see the size explosion.
// And figure out exactly which fields cause the pod grows from
// 720 bytes to 3000 bytes...

// 1.
// Let's do Typemeta first. Additional values are 5 bytes
// Ah... seems like there is no significant change as expected!

// 2.
// Let's do ObjectMeta then... Additional values are around 400 bytes.
// So ideally, now each pod should take 1100 bytes.
// However... It blows up... each pod now actually takes 2000 bytes.
// So where are the 900 bytes come from?
//
// Sure. ObjectMeta contains some non-primitive types like map.
// Is map the real issue? Let's ignore label map, which contains 15 bytes data.
// Wow... Now each pod takes 1600 bytes... So each non-empty map takes 400 bytes!

// Let's count how many maps does a non-empty pod has in the ObjectMeta -> 2!
// OK. Now we learned that additional 900 bytes are from two stupid map structs in go.

// 3.
// Let's do podspec. Additional values are 80 bytes or so.
// We expects the pod to take (720 + 80) = 800 bytes.
// However, now the pod takes 1900 bytes.
// Interesting... Where are the additional 1100 bytes?
// Probably the stupid container array struct?
// Ah... each container has a field called Resources.
// Each Resources has two maps... Each map takes 400 bytes...

// 4.
// Let's do podstatus. Additional values are 7 bytes
// OK. No significant change as expected.

// In conclusion: all the additional memory are took by the map structs in pod.

func main() {
	var (
		tm        bool
		om        bool
		omNoLabel bool
		pspec     bool
		ps        bool
	)
	flag.BoolVar(&tm, "typemeta", false, "fill in typemeta")
	flag.BoolVar(&om, "objectmeta", false, "fill in objectmeta")
	flag.BoolVar(&omNoLabel, "objectmeta-no-label", false, "do not fill in label in objectmeta")
	flag.BoolVar(&pspec, "podspec", false, "fill in podspec")

	flag.BoolVar(&ps, "podstatus", false, "fill in podstatus")

	flag.Parse()

	store = make(map[int]*api.Pod)
	for i := 0; i < 100000; i++ {
		pod := &api.Pod{}
		if tm {
			pod.TypeMeta.Kind = "Pod"
			pod.TypeMeta.APIVersion = "V1"
		}
		if om {
			var data []byte
			if omNoLabel {
				data = omjsonnol
			} else {
				data = omjson
			}
			if err := json.Unmarshal(data, &pod.ObjectMeta); err != nil {
				panic(err)
			}
		}
		if pspec {
			if err := json.Unmarshal(podspecjson, &pod.Spec); err != nil {
				panic(err)
			}
		}
		if ps {
			if err := json.Unmarshal(statusjson, &pod.Status); err != nil {
				panic(err)
			}
		}
		store[i] = pod
	}

	runtime.GC()

	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	fmt.Printf("alloc: %d, sys: %d, idle: %d, inuse: %d\n", st.HeapAlloc, st.HeapSys, st.HeapIdle, st.HeapInuse)
}

var (
	omjson = []byte(`{
      "name":"scale-rc-9-zumj1",
      "generateName":"scale-rc-9-",
      "namespace":"scale-ns-1",
      "uid":"1a332dc8-341f-11e6-84a3-42010af0000e",
      "creationTimestamp":"2016-06-17T00:04:46Z",
      "labels":{
         "name":"scale-label-1-9"
      },
      "annotations":{
         "kubernetes.io/created-by":"{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"scale-ns-1\",\"name\":\"scale-rc-9\",\"uid\":\"e40ad334-341e-11e6-84a3-42010af0000e\",\"apiVersion\":\"v1\",\"resourceVersion\":\"379108\"}}\n"
      }
   }
   `)

	omjsonnol = []byte(`{
      "name":"scale-rc-9-zumj1",
      "generateName":"scale-rc-9-",
      "namespace":"scale-ns-1",
      "uid":"1a332dc8-341f-11e6-84a3-42010af0000e",
      "creationTimestamp":"2016-06-17T00:04:46Z",
      "annotations":{
         "kubernetes.io/created-by":"{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"scale-ns-1\",\"name\":\"scale-rc-9\",\"uid\":\"e40ad334-341e-11e6-84a3-42010af0000e\",\"apiVersion\":\"v1\",\"resourceVersion\":\"379108\"}}\n"
      }
   }
   `)

	podspecjson = []byte(`{
      "containers":[
         {
            "name":"none",
            "image":"none",
            "resources":{},
            "terminationMessagePath":"/dev/termination-log",
            "imagePullPolicy":"Always"
         }
      ],
      "restartPolicy":"Always",
      "terminationGracePeriodSeconds":30,
      "dnsPolicy":"ClusterFirst",
      "securityContext":{}
   }`)

	statusjson = []byte(`{
        "status":{
        "phase":"Pending"
      }
	}`)
)
