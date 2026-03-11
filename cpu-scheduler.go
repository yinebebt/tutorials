package main

import (
	"fmt"
	"sort"
	"strings"
)

// Process represents a process in the scheduling simulation
type Process struct {
	ID             int // Process identifier
	ArrivalTime    int // When process arrives in ready queue
	BurstTime      int // Total CPU time required
	Priority       int // Priority level (lower number = higher priority)
	StartTime      int // When process first gets CPU (-1 if not started)
	CompletionTime int // When process completes execution
	WaitingTime    int // Total time spent waiting in ready queue
	TurnaroundTime int // Total time from arrival to completion
	ResponseTime   int // Time from arrival to first CPU allocation
	RemainingTime  int // Remaining burst time (for preemptive algorithms)
}

// SchedulingResult contains the results of a scheduling algorithm
type SchedulingResult struct {
	Algorithm         string
	Processes         []Process
	GanttChart        []GanttEntry
	AvgWaitingTime    float64
	AvgTurnaroundTime float64
	AvgResponseTime   float64
	TotalTime         int
	ContextSwitches   int
}

// GanttEntry represents one entry in the Gantt chart
type GanttEntry struct {
	ProcessID int
	StartTime int
	EndTime   int
}

// Scheduler interface for different scheduling algorithms
type Scheduler interface {
	Schedule(processes []Process) SchedulingResult
	Name() string
}

// FCFSScheduler implements First Come First Serve scheduling
type FCFSScheduler struct{}

func (f *FCFSScheduler) Name() string {
	return "First Come First Serve (FCFS)"
}

func (f *FCFSScheduler) Schedule(processes []Process) SchedulingResult {
	// Create a copy of processes to avoid modifying original
	procs := make([]Process, len(processes))
	copy(procs, processes)

	// Sort by arrival time
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].ArrivalTime < procs[j].ArrivalTime
	})

	currentTime := 0
	ganttChart := []GanttEntry{}
	contextSwitches := 0

	for i := range procs {
		// If CPU is idle, advance to process arrival time
		if currentTime < procs[i].ArrivalTime {
			currentTime = procs[i].ArrivalTime
		}

		// Set start time (first time getting CPU)
		procs[i].StartTime = currentTime

		// Add to Gantt chart
		ganttChart = append(ganttChart, GanttEntry{
			ProcessID: procs[i].ID,
			StartTime: currentTime,
			EndTime:   currentTime + procs[i].BurstTime,
		})

		// Update current time
		currentTime += procs[i].BurstTime

		// Calculate times
		procs[i].CompletionTime = currentTime
		procs[i].TurnaroundTime = procs[i].CompletionTime - procs[i].ArrivalTime
		procs[i].WaitingTime = procs[i].TurnaroundTime - procs[i].BurstTime
		procs[i].ResponseTime = procs[i].StartTime - procs[i].ArrivalTime

		// Count context switches (all processes except the first one)
		if i > 0 {
			contextSwitches++
		}
	}

	return SchedulingResult{
		Algorithm:         f.Name(),
		Processes:         procs,
		GanttChart:        ganttChart,
		AvgWaitingTime:    calculateAvgWaitingTime(procs),
		AvgTurnaroundTime: calculateAvgTurnaroundTime(procs),
		AvgResponseTime:   calculateAvgResponseTime(procs),
		TotalTime:         currentTime,
		ContextSwitches:   contextSwitches,
	}
}

// SJFScheduler implements Shortest Job First scheduling
type SJFScheduler struct{}

func (s *SJFScheduler) Name() string {
	return "Shortest Job First (SJF)"
}

func (s *SJFScheduler) Schedule(processes []Process) SchedulingResult {
	procs := make([]Process, len(processes))
	copy(procs, processes)

	currentTime := 0
	completed := make([]bool, len(procs))
	ganttChart := []GanttEntry{}
	contextSwitches := 0

	for completedCount := 0; completedCount < len(procs); {
		// Find all processes that have arrived and are not completed
		availableProcesses := []int{}
		for i := range procs {
			if procs[i].ArrivalTime <= currentTime && !completed[i] {
				availableProcesses = append(availableProcesses, i)
			}
		}

		if len(availableProcesses) == 0 {
			// No process available, advance time to next arrival
			nextArrival := int(^uint(0) >> 1) // Max int
			for i := range procs {
				if !completed[i] && procs[i].ArrivalTime > currentTime && procs[i].ArrivalTime < nextArrival {
					nextArrival = procs[i].ArrivalTime
				}
			}
			currentTime = nextArrival
			continue
		}

		// Find process with shortest burst time
		shortestIdx := availableProcesses[0]
		for _, idx := range availableProcesses {
			if procs[idx].BurstTime < procs[shortestIdx].BurstTime {
				shortestIdx = idx
			}
		}

		// Execute the shortest process
		procs[shortestIdx].StartTime = currentTime

		ganttChart = append(ganttChart, GanttEntry{
			ProcessID: procs[shortestIdx].ID,
			StartTime: currentTime,
			EndTime:   currentTime + procs[shortestIdx].BurstTime,
		})

		currentTime += procs[shortestIdx].BurstTime
		procs[shortestIdx].CompletionTime = currentTime
		procs[shortestIdx].TurnaroundTime = procs[shortestIdx].CompletionTime - procs[shortestIdx].ArrivalTime
		procs[shortestIdx].WaitingTime = procs[shortestIdx].TurnaroundTime - procs[shortestIdx].BurstTime
		procs[shortestIdx].ResponseTime = procs[shortestIdx].StartTime - procs[shortestIdx].ArrivalTime

		completed[shortestIdx] = true
		completedCount++

		// Count context switches (all processes except the first one)
		if completedCount > 1 {
			contextSwitches++
		}
	}

	return SchedulingResult{
		Algorithm:         s.Name(),
		Processes:         procs,
		GanttChart:        ganttChart,
		AvgWaitingTime:    calculateAvgWaitingTime(procs),
		AvgTurnaroundTime: calculateAvgTurnaroundTime(procs),
		AvgResponseTime:   calculateAvgResponseTime(procs),
		TotalTime:         currentTime,
		ContextSwitches:   contextSwitches,
	}
}

// RoundRobinScheduler implements Round Robin scheduling
type RoundRobinScheduler struct {
	TimeQuantum int
}

func (r *RoundRobinScheduler) Name() string {
	return fmt.Sprintf("Round Robin (Quantum=%d)", r.TimeQuantum)
}

// Round Robin Implementation
func (r *RoundRobinScheduler) Schedule(processes []Process) SchedulingResult {
	procs := make([]Process, len(processes))
	copy(procs, processes)

	// Initialize remaining time and mark all as not started
	for i := range procs {
		procs[i].RemainingTime = procs[i].BurstTime
		procs[i].StartTime = -1 // Mark as not started
	}

	currentTime := 0
	readyQueue := []int{}
	ganttChart := []GanttEntry{}
	completed := 0
	contextSwitches := 0
	lastProcess := -1
	processAdded := make([]bool, len(procs)) // Track which processes have been added to queue

	for completed < len(procs) {
		// Add newly arrived processes to ready queue
		for i := range procs {
			if procs[i].ArrivalTime <= currentTime && !processAdded[i] && procs[i].RemainingTime > 0 {
				readyQueue = append(readyQueue, i)
				processAdded[i] = true
			}
		}

		if len(readyQueue) == 0 {
			// No process in ready queue, advance to next arrival
			nextArrival := int(^uint(0) >> 1) // Max int
			for i := range procs {
				if procs[i].RemainingTime > 0 && procs[i].ArrivalTime > currentTime && procs[i].ArrivalTime < nextArrival {
					nextArrival = procs[i].ArrivalTime
				}
			}
			if nextArrival == int(^uint(0)>>1) {
				break // No more processes to schedule
			}
			currentTime = nextArrival
			continue
		}

		// Get next process from ready queue
		currentProcIdx := readyQueue[0]
		readyQueue = readyQueue[1:]

		// Set start time if first time getting CPU
		if procs[currentProcIdx].StartTime == -1 {
			procs[currentProcIdx].StartTime = currentTime
		}

		// Count context switch if switching to different process
		if lastProcess != -1 && lastProcess != currentProcIdx {
			contextSwitches++
		}
		lastProcess = currentProcIdx

		// Calculate execution time for this quantum
		execTime := r.TimeQuantum
		if procs[currentProcIdx].RemainingTime < execTime {
			execTime = procs[currentProcIdx].RemainingTime
		}

		// Add to Gantt chart
		ganttChart = append(ganttChart, GanttEntry{
			ProcessID: procs[currentProcIdx].ID,
			StartTime: currentTime,
			EndTime:   currentTime + execTime,
		})

		// Update time and remaining time
		currentTime += execTime
		procs[currentProcIdx].RemainingTime -= execTime

		// Check if process is completed
		if procs[currentProcIdx].RemainingTime == 0 {
			procs[currentProcIdx].CompletionTime = currentTime
			procs[currentProcIdx].TurnaroundTime = procs[currentProcIdx].CompletionTime - procs[currentProcIdx].ArrivalTime
			procs[currentProcIdx].WaitingTime = procs[currentProcIdx].TurnaroundTime - procs[currentProcIdx].BurstTime
			procs[currentProcIdx].ResponseTime = procs[currentProcIdx].StartTime - procs[currentProcIdx].ArrivalTime
			completed++
		} else {
			// Process not completed, add back to ready queue
			readyQueue = append(readyQueue, currentProcIdx)
		}
	}

	return SchedulingResult{
		Algorithm:         r.Name(),
		Processes:         procs,
		GanttChart:        ganttChart,
		AvgWaitingTime:    calculateAvgWaitingTime(procs),
		AvgTurnaroundTime: calculateAvgTurnaroundTime(procs),
		AvgResponseTime:   calculateAvgResponseTime(procs),
		TotalTime:         currentTime,
		ContextSwitches:   contextSwitches,
	}
}

// PriorityScheduler implements Priority scheduling (non-preemptive)
type PriorityScheduler struct{}

func (p *PriorityScheduler) Name() string {
	return "Priority Scheduling (Non-Preemptive)"
}

func (p *PriorityScheduler) Schedule(processes []Process) SchedulingResult {
	procs := make([]Process, len(processes))
	copy(procs, processes)

	currentTime := 0
	completed := make([]bool, len(procs))
	ganttChart := []GanttEntry{}
	contextSwitches := 0

	for completedCount := 0; completedCount < len(procs); {
		// Find all processes that have arrived and are not completed
		availableProcesses := []int{}
		for i := range procs {
			if procs[i].ArrivalTime <= currentTime && !completed[i] {
				availableProcesses = append(availableProcesses, i)
			}
		}

		if len(availableProcesses) == 0 {
			// No process available, advance time to next arrival
			nextArrival := int(^uint(0) >> 1) // Max int
			for i := range procs {
				if !completed[i] && procs[i].ArrivalTime > currentTime && procs[i].ArrivalTime < nextArrival {
					nextArrival = procs[i].ArrivalTime
				}
			}
			currentTime = nextArrival
			continue
		}

		// Find process with highest priority (lowest priority number)
		highestPriorityIdx := availableProcesses[0]
		for _, idx := range availableProcesses {
			if procs[idx].Priority < procs[highestPriorityIdx].Priority {
				highestPriorityIdx = idx
			} else if procs[idx].Priority == procs[highestPriorityIdx].Priority {
				// If same priority, use FCFS (earlier arrival time)
				if procs[idx].ArrivalTime < procs[highestPriorityIdx].ArrivalTime {
					highestPriorityIdx = idx
				}
			}
		}

		// Execute the highest priority process
		procs[highestPriorityIdx].StartTime = currentTime

		ganttChart = append(ganttChart, GanttEntry{
			ProcessID: procs[highestPriorityIdx].ID,
			StartTime: currentTime,
			EndTime:   currentTime + procs[highestPriorityIdx].BurstTime,
		})

		currentTime += procs[highestPriorityIdx].BurstTime
		procs[highestPriorityIdx].CompletionTime = currentTime
		procs[highestPriorityIdx].TurnaroundTime = procs[highestPriorityIdx].CompletionTime - procs[highestPriorityIdx].ArrivalTime
		procs[highestPriorityIdx].WaitingTime = procs[highestPriorityIdx].TurnaroundTime - procs[highestPriorityIdx].BurstTime
		procs[highestPriorityIdx].ResponseTime = procs[highestPriorityIdx].StartTime - procs[highestPriorityIdx].ArrivalTime

		completed[highestPriorityIdx] = true
		completedCount++

		// Count context switches (all processes except the first one)
		if completedCount > 1 {
			contextSwitches++
		}
	}

	return SchedulingResult{
		Algorithm:         p.Name(),
		Processes:         procs,
		GanttChart:        ganttChart,
		AvgWaitingTime:    calculateAvgWaitingTime(procs),
		AvgTurnaroundTime: calculateAvgTurnaroundTime(procs),
		AvgResponseTime:   calculateAvgResponseTime(procs),
		TotalTime:         currentTime,
		ContextSwitches:   contextSwitches,
	}
}

// Helper functions
func calculateAvgWaitingTime(processes []Process) float64 {
	total := 0
	for _, p := range processes {
		total += p.WaitingTime
	}
	return float64(total) / float64(len(processes))
}

func calculateAvgTurnaroundTime(processes []Process) float64 {
	total := 0
	for _, p := range processes {
		total += p.TurnaroundTime
	}
	return float64(total) / float64(len(processes))
}

func calculateAvgResponseTime(processes []Process) float64 {
	total := 0
	for _, p := range processes {
		total += p.ResponseTime
	}
	return float64(total) / float64(len(processes))
}

// Display functions
func printResults(result SchedulingResult) {
	fmt.Printf("\n=== %s ===\n", result.Algorithm)
	fmt.Printf("%-4s %-8s %-8s %-8s %-8s %-12s %-8s %-12s %-12s\n",
		"PID", "Arrival", "Burst", "Priority", "Start", "Completion", "Waiting", "Turnaround", "Response")
	fmt.Println(strings.Repeat("-", 90))

	for _, p := range result.Processes {
		fmt.Printf("%-4d %-8d %-8d %-8d %-8d %-12d %-8d %-12d %-12d\n",
			p.ID, p.ArrivalTime, p.BurstTime, p.Priority, p.StartTime, p.CompletionTime,
			p.WaitingTime, p.TurnaroundTime, p.ResponseTime)
	}

	fmt.Printf("\nPerformance Metrics:\n")
	fmt.Printf("- Average Waiting Time: %.2f\n", result.AvgWaitingTime)
	fmt.Printf("- Average Turnaround Time: %.2f\n", result.AvgTurnaroundTime)
	fmt.Printf("- Average Response Time: %.2f\n", result.AvgResponseTime)
	fmt.Printf("- Total Execution Time: %d\n", result.TotalTime)
	fmt.Printf("- Context Switches: %d\n", result.ContextSwitches)
}

func printGanttChart(result SchedulingResult) {
	fmt.Printf("\nGantt Chart for %s:\n", result.Algorithm)

	// Print process timeline
	fmt.Print("|")
	for _, entry := range result.GanttChart {
		duration := entry.EndTime - entry.StartTime
		processLabel := fmt.Sprintf("P%d", entry.ProcessID)

		if duration == 1 {
			fmt.Printf("P%d|", entry.ProcessID)
		} else if duration == 2 {
			fmt.Printf("P%d|", entry.ProcessID)
		} else if duration >= 3 {
			padding := duration - len(processLabel)
			leftPad := padding / 2
			rightPad := padding - leftPad
			fmt.Printf("%s%s%s|", strings.Repeat(" ", leftPad), processLabel, strings.Repeat(" ", rightPad))
		}
	}
	fmt.Println()

	// Print time markers with proper spacing
	fmt.Print("0")
	for _, entry := range result.GanttChart {
		duration := entry.EndTime - entry.StartTime
		endTimeStr := fmt.Sprintf("%d", entry.EndTime)
		spaces := duration - len(endTimeStr)
		if spaces < 0 {
			spaces = 0
		}
		fmt.Printf("%s%s", strings.Repeat(" ", spaces), endTimeStr)
	}
	fmt.Println("\n" + strings.Repeat("-", 50))
}

func main() {
	// Sample process set for demonstration
	// P1: Long CPU-bound task (video encoding, scientific computation)
	// P2: Medium interactive task (user application)
	// P3: Long batch job (database backup, file processing)
	// P4: Medium background task (system maintenance)
	// P5: Short interactive task (file operation, system query)
	processes := []Process{
		{ID: 1, ArrivalTime: 0, BurstTime: 8, Priority: 2}, // Long CPU-bound
		{ID: 2, ArrivalTime: 1, BurstTime: 4, Priority: 1}, // Medium interactive
		{ID: 3, ArrivalTime: 2, BurstTime: 9, Priority: 3}, // Long batch job
		{ID: 4, ArrivalTime: 3, BurstTime: 5, Priority: 2}, // Medium background
		{ID: 5, ArrivalTime: 4, BurstTime: 2, Priority: 1}, // Short interactive
	}

	fmt.Println("CPU Scheduling Simulation")
	fmt.Printf("Processes: %d | Total Burst Time: %d units\n", len(processes), getTotalBurstTime(processes))
	fmt.Println("Process Mix: Long CPU-bound, Interactive, Batch, Background tasks")
	fmt.Println()

	// Create schedulers with consistent context switch counting
	schedulers := []Scheduler{
		&FCFSScheduler{},
		&SJFScheduler{},
		&RoundRobinScheduler{TimeQuantum: 3},
		&PriorityScheduler{},
	}

	// Store results for comparison
	results := make([]SchedulingResult, len(schedulers))

	// Run each scheduling algorithm
	for i, scheduler := range schedulers {
		results[i] = scheduler.Schedule(processes)
		printResults(results[i])
		printGanttChart(results[i])
		fmt.Println()
	}

	// Performance comparison table
	fmt.Println("\nPerformance Comparison Summary")
	fmt.Printf("%-30s %-12s %-12s %-12s %-10s\n", "Algorithm", "Avg Wait", "Avg TAT", "Avg Response", "Ctx Switch")
	fmt.Println(strings.Repeat("-", 85))

	for _, result := range results {
		fmt.Printf("%-30s %-12.2f %-12.2f %-12.2f %-10d\n",
			result.Algorithm, result.AvgWaitingTime, result.AvgTurnaroundTime,
			result.AvgResponseTime, result.ContextSwitches)
	}

	// Find best algorithms for different metrics
	fmt.Println("\nBest Performance Categories:")
	bestWait := findBestAlgorithm(results, "waiting")
	bestTAT := findBestAlgorithm(results, "turnaround")
	bestResponse := findBestAlgorithm(results, "response")

	fmt.Printf("• Best Average Waiting Time: %s (%.2f)\n", bestWait.Algorithm, bestWait.AvgWaitingTime)
	fmt.Printf("• Best Average Turnaround Time: %s (%.2f)\n", bestTAT.Algorithm, bestTAT.AvgTurnaroundTime)
	fmt.Printf("• Best Average Response Time: %s (%.2f)\n", bestResponse.Algorithm, bestResponse.AvgResponseTime)
}

func getTotalBurstTime(processes []Process) int {
	total := 0
	for _, p := range processes {
		total += p.BurstTime
	}
	return total
}

func findBestAlgorithm(results []SchedulingResult, metric string) SchedulingResult {
	best := results[0]

	for _, result := range results[1:] {
		switch metric {
		case "waiting":
			if result.AvgWaitingTime < best.AvgWaitingTime {
				best = result
			}
		case "turnaround":
			if result.AvgTurnaroundTime < best.AvgTurnaroundTime {
				best = result
			}
		case "response":
			if result.AvgResponseTime < best.AvgResponseTime {
				best = result
			}
		}
	}

	return best
}
