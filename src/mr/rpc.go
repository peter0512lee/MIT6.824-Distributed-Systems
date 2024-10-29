package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type TaskCompletedStatus int

const (
	MapTaskCompleted = iota
	MapTaskFailed
	ReduceTaskCompleted
	ReduceTaskFailed
)

type TaskType int

const (
	MapTask = iota
	ReduceTask
	Wait
	Exit
)

type MessageSend struct {
	TaskID              int                 // task id
	TaskCompletedStatus TaskCompletedStatus // task completed status
}

type MessageReply struct {
	TaskID   int      // task id
	TaskType TaskType // task type, map or reduce or wait or exit
	TaskFile string   // task file name
	NReduce  int      // reduce number, indicate the number of reduce tasks
	NMap     int      // map number, indicate the number of map tasks
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
