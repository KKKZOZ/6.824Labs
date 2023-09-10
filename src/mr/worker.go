package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

var M int = 0
var N int = 0
var Id int = 0

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// get basic info of current job
	getBasicInfo()

	// Endless for loop
EndlessLoop:
	for {
		time.Sleep(1 * time.Second)
		taskInfo, err := getTaskInfo()
		if err != nil {
			log.Printf("Worker %v: exiting for unknown reason...", Id)
			break
		}

		log.Printf("Worker %v: Get Task: %v\n", Id, taskInfo)

		switch taskInfo.TaskType {
		case "Wait":
			continue
		case "Done":
			break EndlessLoop
		case "Map":
			execMapTask(taskInfo, mapf)
		case "Reduce":
			execReduceTask(taskInfo, reducef)
		}

		// notify coordinator that task is done
		TaskDone(taskInfo)
	}

}

func TaskDone(taskInfo TaskInfo) {
	log.Printf("Worker %v: TaskDone: %v\n", Id, taskInfo)
	args := taskInfo
	reply := ExampleReply{}
	ok := call("Coordinator.TaskDone", &args, &reply)
	if !ok {
		log.Fatalf("TaskDone Error")
	}
}

func execReduceTask(taskInfo TaskInfo, reducef func(string, []string) string) {

	res := []KeyValue{}
	// read all files according to taskInfo
	for i := 0; i < M; i++ {
		fileName := fmt.Sprintf("mr-%d-%d", i, taskInfo.TaskId)
		file, err := os.Open(fileName)
		if err != nil {
			log.Fatalf("Worker %v: cannot open %v", Id, fileName)
		}

		dec := json.NewDecoder(file)

		var kvList []KeyValue
		if err := dec.Decode(&kvList); err != nil {
			break
		}
		res = append(res, kvList...)

		file.Close()
	}
	// Sort
	sort.Sort(ByKey(res))

	oname := fmt.Sprintf("mr-out-%d", taskInfo.TaskId)
	ofile, _ := os.Create(oname)
	i := 0
	for i < len(res) {
		j := i + 1
		for j < len(res) && res[j].Key == res[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, res[k].Value)
		}
		output := reducef(res[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", res[i].Key, output)

		i = j
	}

	ofile.Close()

}

func execMapTask(taskInfo TaskInfo, mapf func(string, string) []KeyValue) {
	fileName := taskInfo.FileName
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("cannot open %v", fileName)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", fileName)
	}
	file.Close()
	kva := mapf(fileName, string(content))
	intermediate := []KeyValue{}
	intermediate = append(intermediate, kva...)

	// process intermediate data, store them into NReduce files
	for i := 0; i < N; i++ {
		fileName := fmt.Sprintf("mr-%d-%d", taskInfo.TaskId, i)
		file, err := os.Create(fileName)
		if err != nil {
			log.Fatalf("cannot open %v", fileName)
		}
		defer file.Close()

		res := []KeyValue{}
		for _, kv := range intermediate {
			if ihash(kv.Key)%N == i {
				res = append(res, kv)
			}
		}

		// Write to file in json
		json.NewEncoder(file).Encode(res)

	}
}

func getTaskInfo() (TaskInfo, error) {
	args := ExampleArgs{}
	reply := TaskInfo{}
	ok := call("Coordinator.GetTask", &args, &reply)
	if ok {
		return reply, nil
	} else {
		return reply, fmt.Errorf("Get Task Error")
	}

}

func getBasicInfo() {
	args := ExampleArgs{}
	reply := BasicInfo{}

	ok := call("Coordinator.GetBasicInfo", &args, &reply)
	if ok {
		M = reply.M
		N = reply.N
		Id = reply.Id
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Printf("Worker %v: dialing:%v\n", Id, err)
		return false
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
