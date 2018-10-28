package mapreduce

import (
	"fmt"
	"sync"
)

//
// schedule() starts and waits for all tasks in the given phase (mapPhase
// or reducePhase). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var n_other int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mapFiles)
		n_other = nReduce
	case reducePhase:
		ntasks = nReduce
		n_other = len(mapFiles)
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, n_other)

	// All ntasks tasks have to be scheduled on workers. Once all tasks
	// have completed successfully, schedule() should return.
	//
	// Your code here (Part III, Part IV).

	args := DoTaskArgs{}
	args.JobName = jobName
	args.Phase = phase
	args.NumOtherPhase = n_other
	closeChannel := make(chan struct{})
	var wait sync.WaitGroup
	taskList := make(chan DoTaskArgs, ntasks)
	// 隐式假设ntasks比worker的数量要多
	workerChannel := make(chan string, ntasks)
	wait.Add(ntasks)

	go func() {
		for {
			workerAddress := <-registerChan
			workerChannel <- workerAddress
		}
	}()

	go func(waitPtr *sync.WaitGroup) {
		waitPtr.Wait()
		close(closeChannel)
	}(&wait)

	for i := 0; i < ntasks; i++ {
		if args.Phase == mapPhase {
			args.File = mapFiles[i]
		}
		args.TaskNumber = i
		taskList <- args
	}

	go func() {
		for {
			workerAddress := <-workerChannel
			innerArgs := <-taskList
			go func(waitPtr *sync.WaitGroup, args DoTaskArgs, address string) {
				ret := call(address, "Worker.DoTask", innerArgs, nil)
				if !ret {
					taskList <- args
					return
				}
				waitPtr.Done()
				workerChannel <- address
			}(&wait, innerArgs, workerAddress)
		}
	}()

	<-closeChannel
	fmt.Printf("Schedule: %v done\n", phase)
}
