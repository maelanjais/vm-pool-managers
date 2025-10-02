package worker

import (
	"PoolManagerVM/backend/internal/jobs"
	"PoolManagerVM/backend/models"
	"context"
	"fmt"
	"log"
	"sync"
)

var (
	HighPriorityJobs   chan models.Job
	NormalPriorityJobs chan models.Job
)

// LaunchWorkers starts a pool of worker goroutines to process jobs.
//
// Parameters:
//   - numWorkers: The number of worker goroutines to launch.
//   - wg:         Pointer to a WaitGroup to track when all workers finish.
//   - ctx:        Context to allow cancellation of all workers.
//
// It initializes two channels for job queues:
//   - HighPriorityJobs: buffered channel with capacity 50
//   - NormalPriorityJobs: buffered channel with capacity 100
//
// Each worker listens for jobs on both channels and processes them until the context is cancelled.
func LaunchWorkers(numWorkers int, wg *sync.WaitGroup, ctx context.Context) {
	HighPriorityJobs = make(chan models.Job, 50)
	NormalPriorityJobs = make(chan models.Job, 100)

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, wg, ctx)
	}
}

// worker continuously listens for jobs from HighPriorityJobs and NormalPriorityJobs channels.
//
// Parameters:
//   - id: Worker ID used for logging.
//   - wg: WaitGroup to mark worker as done when it exits.
//   - ctx: Context used to stop the worker gracefully.
//
// Workflow:
//  1. Listens for context cancellation to stop.
//  2. Reads from high-priority channel first, then normal-priority channel.
//  3. Processes each job using processJob.
//  4. Handles channel closure gracefully.
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

// processJob executes the logic for a single job based on its type.
//
// Parameters:
//   - workerID: The ID of the worker processing the job, used for logging.
//   - job:      The Job struct containing type and data.
//
// Currently supported job types:
//   - models.CreateVM: calls jobs.CreateVM to create a VM.
//   - models.DeleteVM: calls jobs.DeleteVM to delete a VM instance, logging errors if deletion fails.
func processJob(workerID int, job models.Job) {
	switch job.Type {
	case models.CreateVM:
		jobs.CreateVM(workerID, job)

	case models.AttribVM:
		jobs.AttribVM(workerID, job)

	case models.DeleteVM:
		instanceID := job.Data["instance_id"]
		err := jobs.DeleteVM(instanceID)
		if err != nil {
			log.Println("Failed to delete VM:", err)
		} else {
			log.Println("VM deleted successfully:", instanceID)
		}
	case models.CreateVolumeAndAttach:
		err := jobs.CreateVolumeAndAttach(workerID, job)
		if err != nil {
			log.Println("Failed to create and attach volume:", err)
		} else {
			log.Println("Volume created and attached successfully")
		}

	case models.DeleteVolume:
		instanceID := job.Data["instance_id"]
		err := jobs.DeleteVolume(instanceID)
		if err != nil {
			log.Println("Failed to delete Volume:", err)
		} else {
			log.Println("Volume deleted successfully:", instanceID)
		}
	}
}

// CreateJob creates a new Job struct with the given type and data.
//
// Parameters:
//   - JobType: The type of job (CreateVM, DeleteVM, etc.).
//   - data:    A map containing the job parameters.
//
// Returns:
//   - Pointer to the newly created Job struct.
func CreateJob(JobType models.JobType, data map[string]string) *models.Job {
	return &models.Job{
		Type: JobType,
		Data: data,
	}
}

// AddJob adds a job to the appropriate job queue (high or normal priority).
//
// Parameters:
//   - job:          The Job struct to enqueue.
//   - highPriority: If true, adds the job to HighPriorityJobs, otherwise to NormalPriorityJobs.
//
// Notes:
//   - This function blocks if the channel buffer is full until a worker reads a job.

func AddJob(job models.Job, highPriority bool) {
	fmt.Printf("Adding job to queue\n")
	if highPriority {
		HighPriorityJobs <- job
	} else {
		NormalPriorityJobs <- job
	}
}
