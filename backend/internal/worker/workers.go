package worker

import (
	"PoolManagerVM/backend/utils"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type JobType int

const (
	CreateVM JobType = iota
	CreateVMAdmin
	DeleteVM
	AttribVM
)

type Job struct {
	ID   uint64
	Name string
	Type JobType
	Data map[string]string
	// retryCount int
}

var HighPriorityJobs chan Job
var NormalPriorityJobs chan Job

func LaunchWorkers(numWorkers int, wg *sync.WaitGroup, ctx context.Context) {
	HighPriorityJobs = make(chan Job, 50)
	NormalPriorityJobs = make(chan Job, 100)

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, wg, ctx)
	}
}

func worker(id int, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping\n", id)
			return
		case job, ok := <-HighPriorityJobs:
			if !ok {
				HighPriorityJobs = nil
				continue
			}
			processJob(id, job)
		case job, ok := <-NormalPriorityJobs:
			if !ok {
				NormalPriorityJobs = nil
				continue
			}
			processJob(id, job)
		}
	}
}

func processJob(workerID int, job Job) {
	if job.Type == CreateVMAdmin {
		cfg, err := utils.LoadConfig("config.toml")
		if err != nil {
			log.Printf("Error\n")
			return
		}
		fmt.Println("Worker ", workerID, " takes the job of creating a base model VM")
		CreateVMbase(*cfg)
		fmt.Println("Worker ", workerID, " finished its job")
	} else {
		// sleep
		time.Sleep(time.Second)
	}
}

var jobCount uint64

func CreateJob(name string, JobType JobType, data map[string]string) *Job {
	jobCount++
	return &Job{
		ID:   jobCount,
		Name: name,
		Type: JobType,
		Data: data,
	}
}

func AddJob(job Job, highPriority bool) {
	fmt.Printf("Adding job %d to queue\n", job.ID)
	if highPriority {
		HighPriorityJobs <- job
	} else {
		NormalPriorityJobs <- job
	}
}
