package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
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

	// store states of map tasks
	mapTaskList []TaskState
	// store states of reduce tasks
	reduceTaskList []TaskState

	files []string
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) GetBasicInfo(args *ExampleArgs, reply *BasicInfo) error {
	reply.M = c.M
	reply.N = c.N
	return nil
}

func (c *Coordinator) GetTask(args *ExampleArgs, reply *TaskInfo) error {
	if c.completedMapCnt < c.M {
		reply.TaskType = "Map"
		id := c.GetIdleTask(c.mapTaskList)
		reply.TaskId = id
		if id == -1 {
			return nil
		}
		reply.FileName = c.files[id]

		// TODO : lock
		go func() {
			c.mapTaskList[id].State = 1
			time.Sleep(10 * time.Second)
			if c.mapTaskList[id].State == 1 {
				c.mapTaskList[id].State = 0
			}
		}()
	} else {
		reply.TaskType = "Reduce"
		id := c.GetIdleTask(c.reduceTaskList)
		reply.TaskId = id
		if id == -1 {
			return nil
		}

		// TODO : lock
		go func() {
			c.reduceTaskList[id].State = 1
			time.Sleep(10 * time.Second)
			if c.reduceTaskList[id].State == 1 {
				c.reduceTaskList[id].State = 0
			}
		}()

	}

	return nil
}

func (c *Coordinator) TaskDone(args *TaskInfo, reply *ExampleReply) error {
	id := args.TaskId
	if args.TaskType == "Map" {
		// only if the task is in progress
		if c.mapTaskList[id].State == 1 {
			c.mapTaskList[id].State = 2
			c.completedMapCnt++
		}
	} else {
		// only if the task is in progress
		if c.reduceTaskList[id].State == 1 {
			c.reduceTaskList[id].State = 2
			c.completedReduceCnt++
		}
	}
	return nil
}

func (c *Coordinator) GetIdleTask(list []TaskState) int {
	for idx, state := range list {
		if state.State == 0 {
			return idx
		}
	}
	return -1
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	fmt.Println("server start...")
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

	if c.completedReduceCnt == c.N {
		return true
	} else {
		return false
	}

}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		M:                  len(files),
		N:                  nReduce,
		completedMapCnt:    0,
		completedReduceCnt: 0,
		mapTaskList:        make([]TaskState, len(files)),
		reduceTaskList:     make([]TaskState, nReduce),
		files:              files,
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
