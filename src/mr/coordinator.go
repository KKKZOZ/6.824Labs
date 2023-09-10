package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type TaskState struct {
	Id    int
	State int
}

type Coordinator struct {
	// the number of Map Task
	M int
	// the number of Reduce Task
	N int

	// the number fo completed Map Task
	completedMapCnt int

	// the number fo completed Reduce Task
	completedReduceCnt int

	// the mutex for cnt
	cntMutex sync.Mutex

	// store states of map tasks
	mapTaskList []TaskState
	// store states of reduce tasks
	reduceTaskList []TaskState

	// the mutex for taskList
	listMutex sync.Mutex

	// to determine whether the job is done
	completed WithMutex[bool]

	// file list
	files []string

	// for better debug experience

	registedWorkerCnt WithMutex[int]
}

type WithMutex[T any] struct {
	mu   sync.Mutex
	data T
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) GetBasicInfo(args *ExampleArgs, reply *BasicInfo) error {
	reply.M = c.M
	reply.N = c.N
	c.registedWorkerCnt.mu.Lock()
	defer c.registedWorkerCnt.mu.Unlock()
	c.registedWorkerCnt.data++
	reply.Id = c.registedWorkerCnt.data
	return nil
}

// Set updates the state of a map or reduce task in the coordinator's task list.
// taskType specifies the type of task list (either "mapTaskList" or "reduceTaskList").
// id specifies the id of the task to be updated.
// state specifies the new state of the task.
func (c *Coordinator) Set(taskType string, id int, state int) {
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	if taskType == "mapTaskList" {
		c.mapTaskList[id].State = state
	}
	if taskType == "reduceTaskList" {
		c.reduceTaskList[id].State = state
	}
}

// Get returns the state of a map or reduce task with the given id.
// The taskType parameter should be either "mapTaskList" or "reduceTaskList".
// Returns 0 if the taskType is invalid or the id is out of range.
func (c *Coordinator) Get(taskType string, id int) int {
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	if taskType == "mapTaskList" {
		return c.mapTaskList[id].State
	}
	if taskType == "reduceTaskList" {
		return c.reduceTaskList[id].State
	}
	return 0
}

func (c *Coordinator) GetTask(args *ExampleArgs, reply *TaskInfo) error {
	c.cntMutex.Lock()
	defer c.cntMutex.Unlock()

	if c.completedReduceCnt == c.N {
		reply.TaskType = "Done"
		return nil
	}

	if c.completedMapCnt < c.M {
		reply.TaskType = "Map"
		id := c.GetIdleTask("Map")

		reply.TaskId = id
		if id == -1 {
			reply.TaskType = "Wait"
			return nil
		}
		reply.FileName = c.files[id]

		// wait for 10 seconds then check
		go func() {
			time.Sleep(10 * time.Second)
			if c.Get("mapTaskList", id) == 1 {
				c.Set("mapTaskList", id, 0)
				log.Printf("Master: Map Task %v Timeout, Reschedule...\n", id)
			}
		}()
	} else {
		reply.TaskType = "Reduce"
		id := c.GetIdleTask("Reduce")
		reply.TaskId = id
		if id == -1 {
			reply.TaskType = "Wait"
			return nil
		}
		// wait for 10 seconds then check
		go func() {
			time.Sleep(10 * time.Second)
			if c.Get("reduceTaskList", id) == 1 {
				c.Set("reduceTaskList", id, 0)
				log.Printf("Master: Reduce Task %v Timeout, Reschedule...\n", id)
			}
		}()

	}

	return nil
}

func (c *Coordinator) TaskDone(args *TaskInfo, reply *ExampleReply) error {
	id := args.TaskId
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	c.cntMutex.Lock()
	defer c.cntMutex.Unlock()
	if args.TaskType == "Map" {
		// only if the task is in progress
		if c.mapTaskList[id].State == 1 {
			c.mapTaskList[id].State = 2
			c.completedMapCnt++
		}

		log.Println("Master: Map Task Done: ", id, c.completedMapCnt, c.M)
		log.Println("Master: ", c.mapTaskList)

	} else {
		// only if the task is in progress
		if c.reduceTaskList[id].State == 1 {
			c.reduceTaskList[id].State = 2
			c.completedReduceCnt++

			if c.completedReduceCnt == c.N {
				log.Println("Master: All tasks are done")
				go func() {
					time.Sleep(5 * time.Second)
					c.completed.mu.Lock()
					c.completed.data = true
					c.completed.mu.Unlock()
				}()

			}

		}
	}
	return nil
}

func (c *Coordinator) GetIdleTask(task string) int {
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	var list []TaskState
	if task == "Map" {
		list = c.mapTaskList
	}
	if task == "Reduce" {
		list = c.reduceTaskList
	}
	id := -1
	for idx, state := range list {
		if state.State == 0 {
			id = idx
			break
		}
	}
	if id != -1 {
		if task == "Map" {
			c.mapTaskList[id].State = 1
		}
		if task == "Reduce" {
			c.reduceTaskList[id].State = 1
		}
	}
	return id
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.completed.mu.Lock()
	defer c.completed.mu.Unlock()
	return c.completed.data
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	log.Printf("Master: MakeCoordinator: M: %v N: %v\n", len(files), nReduce)

	c := Coordinator{
		M:                  len(files),
		N:                  nReduce,
		completedMapCnt:    0,
		completedReduceCnt: 0,
		mapTaskList:        make([]TaskState, len(files)),
		reduceTaskList:     make([]TaskState, nReduce),
		files:              files,
		completed:          WithMutex[bool]{data: false, mu: sync.Mutex{}},
		registedWorkerCnt:  WithMutex[int]{data: 0, mu: sync.Mutex{}},
	}

	// Your code here.
	for idx, item := range c.mapTaskList {
		item.State = 0
		item.Id = idx
	}
	for idx, item := range c.reduceTaskList {
		item.State = 0
		item.Id = idx
	}

	c.server()
	return &c
}
