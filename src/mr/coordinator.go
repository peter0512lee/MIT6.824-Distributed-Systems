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

type TaskStatus int

const (
	Unassigned = iota
	Assigned
	Completed
	Failed
)

type TaskInfo struct {
	TaskStatus TaskStatus // task status
	TaskFile   string     // task file
	TimeStamp  time.Time  // time stamp, indicating the running time of the task
}

type Coordinator struct {
	NMap                   int        // number of map tasks
	NReduce                int        // number of reduce tasks
	MapTasks               []TaskInfo // map task
	ReduceTasks            []TaskInfo // reduce task
	AllMapTaskCompleted    bool       // whether all map tasks have been completed
	AllReduceTaskCompleted bool       // whether all reduce tasks have been completed
	Mutex                  sync.Mutex // mutex, used to protect the shared data
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) InitTask(file []string) {
	for idx := range file {
		c.MapTasks[idx] = TaskInfo{
			TaskFile:   file[idx],
			TaskStatus: Unassigned,
			TimeStamp:  time.Now(),
		}
	}
	for idx := range c.ReduceTasks {
		c.ReduceTasks[idx] = TaskInfo{
			TaskStatus: Unassigned,
		}
	}
}

func (c *Coordinator) RequestTask(args *MessageSend, reply *MessageReply) error {
	// lock
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	// assign map task
	if !c.AllMapTaskCompleted {
		// count the number of completed map tasks
		NMapTaskCompleted := 0
		for idx, taskInfo := range c.MapTasks {
			if taskInfo.TaskStatus == Unassigned || taskInfo.TaskStatus == Failed ||
				(taskInfo.TaskStatus == Assigned && time.Since(taskInfo.TimeStamp) > 10*time.Second) {
				reply.TaskFile = taskInfo.TaskFile
				reply.TaskID = idx
				reply.TaskType = MapTask
				reply.NReduce = c.NReduce
				reply.NMap = c.NMap
				c.MapTasks[idx].TaskStatus = Assigned  // mark the task as assigned
				c.MapTasks[idx].TimeStamp = time.Now() // update the time stamp
				return nil
			} else if taskInfo.TaskStatus == Completed {
				NMapTaskCompleted++
			}
		}
		// check if all map tasks have been completed
		if NMapTaskCompleted == len(c.MapTasks) {
			c.AllMapTaskCompleted = true
		} else {
			reply.TaskType = Wait
			return nil
		}
	}

	// assign reduce task
	if !c.AllReduceTaskCompleted {
		// count the number of completed reduce tasks
		NReduceTaskCompleted := 0
		for idx, taskInfo := range c.ReduceTasks {
			if taskInfo.TaskStatus == Unassigned || taskInfo.TaskStatus == Failed ||
				(taskInfo.TaskStatus == Assigned && time.Since(taskInfo.TimeStamp) > 10*time.Second) {
				reply.TaskID = idx
				reply.TaskType = ReduceTask
				reply.NReduce = c.NReduce
				reply.NMap = c.NMap
				c.ReduceTasks[idx].TaskStatus = Assigned  // mark the task as assigned
				c.ReduceTasks[idx].TimeStamp = time.Now() // update the time stamp
				return nil
			} else if taskInfo.TaskStatus == Completed {
				NReduceTaskCompleted++
			}
		}
		// check if all reduce tasks have been completed
		if NReduceTaskCompleted == len(c.ReduceTasks) {
			c.AllReduceTaskCompleted = true
		} else {
			reply.TaskType = Wait
			return nil
		}
	}

	// all tasks have been completed
	reply.TaskType = Exit
	return nil
}

func (c *Coordinator) ReportTask(args *MessageSend, reply *MessageReply) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if args.TaskCompletedStatus == MapTaskCompleted {
		c.MapTasks[args.TaskID].TaskStatus = Completed
		return nil
	} else if args.TaskCompletedStatus == MapTaskFailed {
		c.MapTasks[args.TaskID].TaskStatus = Failed
		return nil
	} else if args.TaskCompletedStatus == ReduceTaskCompleted {
		c.ReduceTasks[args.TaskID].TaskStatus = Completed
		return nil
	} else if args.TaskCompletedStatus == ReduceTaskFailed {
		c.ReduceTasks[args.TaskID].TaskStatus = Failed
		return nil
	}
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
// func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
// 	reply.Y = args.X + 1
// 	return nil
// }

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
	// confirm the all map tasks have been completed
	for _, taskInfo := range c.MapTasks {
		if taskInfo.TaskStatus != Completed {
			return false
		}
	}
	// confirm the all reduce tasks have been completed
	for _, taskInfo := range c.ReduceTasks {
		if taskInfo.TaskStatus != Completed {
			return false
		}
	}
	return true
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		NReduce:                nReduce,
		NMap:                   len(files),
		MapTasks:               make([]TaskInfo, len(files)),
		ReduceTasks:            make([]TaskInfo, nReduce),
		AllMapTaskCompleted:    false,
		AllReduceTaskCompleted: false,
		Mutex:                  sync.Mutex{},
	}
	c.InitTask(files)

	c.server()
	return &c
}
