/*
 * @Author: shgopher shgopher@gmail.com
 * @Date: 2024-01-13 16:56:31
 * @LastEditors: shgopher shgopher@gmail.com
 * @LastEditTime: 2024-01-13 17:01:24
 * @FilePath: /grpool/examples/first.go
 * @Description:
 *
 * Copyright (c) 2024 by shgopher, All Rights Reserved.
 */
package grpool

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shgopher/grpool"
)

func first() {
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	// number of workers, and size of job queue
	pool := grpool.NewPool(100, 50)

	// release resources used by pool
	defer pool.Release()

	// submit one or more jobs to pool
	for i := 0; i < 10; i++ {
		count := i

		pool.JobQueue <- func() {
			fmt.Printf("I am worker! Number %d\n", count)
		}
	}

	// dummy wait until jobs are finished
	time.Sleep(1 * time.Second)
}
