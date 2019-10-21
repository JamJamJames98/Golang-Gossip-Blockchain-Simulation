package main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"strconv"
	"runtime"
	"math/rand"
	"time"
	"sync/atomic"
	"runtime/debug"
)

type node struct {
	id int
	version int
	status bool
	channel chan int
	requestChannel chan RequestData
	neighbourNodes []*node
	neighbourIndices []int
	neighboursSentTo []int
	numberOfMessages int
	numberOfUpdates int
	requestData RequestData
	nodeType string
}

type RequestData struct {
	channel chan int
	version int
}

type networkStatus struct {
	version int
	count int
}

var gossiping *int32 = new(int32)
var consensus *int32 = new(int32)
var updateBackLog *int32 = new(int32)
var randomUpdateInterval *int32 = new(int32)

/*
 *	Spawns a node
 */
func runNode(id int, name string, myNode *node, size int) {
	runtime.Gosched()
	for runtime.NumGoroutine() < size {
		time.Sleep(1 * time.Nanosecond)
	}

	for {
		runtime.Gosched()
		afterChannel := make(<-chan time.Time)

		//only implement the AfterChannel if it is a node that can pull
		if (*myNode).nodeType == "PUSH&PULL" {
			randTime := rand.Intn(int(atomic.LoadInt32(randomUpdateInterval)))
			afterChannel = time.After(time.Duration(randTime) * time.Second)
		}
		
		time.Sleep(1 * time.Nanosecond)
		select {
		case recievedValue := <- (*myNode).channel:
			atomic.AddInt32(gossiping, 1)
			if recievedValue == -1 {
				atomic.AddInt32(gossiping, -1)
				return
			} else if recievedValue > 0 {	
				fmt.Println("Node id", id, "recieved a value")
				sleepDuration := time.Duration(rand.Intn(600 - 40 + 1) + 40) * time.Millisecond
				time.Sleep(sleepDuration)
				if recievedValue >= (*myNode).version {	
					if recievedValue > (*myNode).version {
						atomic.AddInt32(consensus, 1)
						(*myNode).version = recievedValue
						for i := 0; i < len((*myNode).neighbourNodes); i++ {
							if (*myNode).neighbourNodes[i].channel != (*myNode).channel {
								atomic.AddInt32(updateBackLog, 1)
								(*myNode).neighbourNodes[i].channel <- (*myNode).version
								(*myNode).numberOfUpdates++
							}
						}
						fmt.Println("Finished sending to neighbours")
					} else {
					}
				} else {	
				}
			}
			atomic.AddInt32(updateBackLog, -1)
			atomic.AddInt32(gossiping, -1)
		case <- afterChannel:
			fmt.Println("AFter channel worked")
			neighbourToRequestFrom := rand.Intn(len((*myNode).neighbourNodes))
			(*myNode).requestData.version = (*myNode).version
			(*myNode).neighbourNodes[neighbourToRequestFrom].requestChannel <- (*myNode).requestData
		case recievedData := <- (*myNode).requestChannel:
			sleepDuration := time.Duration(rand.Intn(600 - 40 + 1) + 40) * time.Millisecond
			time.Sleep(sleepDuration)
			if recievedData.version < (*myNode).version {
				atomic.AddInt32(updateBackLog, 1)
				recievedData.channel <- (*myNode).version
				(*myNode).numberOfUpdates++
			}
		}
	}
}

/*
 *	Provides updates to the user when spawning/killing nodes
 */
func progressUpdate(i int, size int, prev float64, message string) float64 {
	exactPercentage := float64(i)/float64(size)*float64(100)
	truncatedPercentage := float64(int(exactPercentage * 100) / 100)
	if prev != truncatedPercentage {	
		fmt.Print(message,truncatedPercentage,"%","\n")
		prev = truncatedPercentage
	}
	return prev
}

/*
 *	Orchestrator method to spawn waiting threads, calls runNode
 */
func spawnFunction(size int, nodes *[]node, neighbourListSize int, nodeType string) {
	(*nodes) = make([]node, size)
	prev := float64(0)
	for i := 0; i < size; i++ {
		prev = progressUpdate(i, size, prev, "Spawn Progress: ")
		(*nodes)[i].channel = make(chan int, size+1)
		(*nodes)[i].requestChannel = make(chan RequestData, size+1)
		(*nodes)[i].requestData.channel = (*nodes)[i].channel
		(*nodes)[i].id = i
		(*nodes)[i].version = 0
		(*nodes)[i].nodeType = nodeType
		if (neighbourListSize > size) {
			neighbourIndices := rand.Perm(size)[:size]
			(*nodes)[i].neighbourIndices = neighbourIndices
			(*nodes)[i].neighbourNodes = make([]*node, len(neighbourIndices))
		} else {
			neighbourIndices := rand.Perm(size)[:neighbourListSize]
			(*nodes)[i].neighbourIndices = neighbourIndices
			(*nodes)[i].neighbourNodes = make([]*node, len(neighbourIndices))
		}
		for j := 0; j < len((*nodes)[i].neighbourIndices); j++ {
			(*nodes)[i].neighbourNodes[j] = &(*nodes)[(*nodes)[i].neighbourIndices[j]]
		}
		go runNode(i, "Spawning", &(*nodes)[i], size)
	}
}

/*
 *	Orchestrator method to kill all waiting threads, calls killNode
 */
func killRoutines(nodes *[]node) {
	for i := 0; i < len(*nodes); i++ {
		(*nodes)[i].channel <- -1
	}
	for runtime.NumGoroutine() > 1 {
		time.Sleep(1 * time.Nanosecond)
	}
	(*nodes) = nil
	atomic.AddInt32(gossiping, -atomic.LoadInt32(gossiping))
    atomic.AddInt32(consensus, -atomic.LoadInt32(consensus))
    atomic.AddInt32(updateBackLog, -atomic.LoadInt32(updateBackLog))
    runtime.GC()
    debug.FreeOSMemory()
}

func readFromFile() []string {
	loadedCommands := make([]string, 0)
	fileToRead, error := os.Open("../../commands.txt")
    if error != nil {
        fmt.Println("No Config file provided")
        fileToRead.Close()
        return loadedCommands
    }    

    scanner := bufio.NewScanner(fileToRead)
    for scanner.Scan() {
    	if len(scanner.Text()) > 0 {
    		    	loadedCommands = append(loadedCommands, scanner.Text())
    	}
    }

    if error := scanner.Err(); error != nil {
        fmt.Println("Fatal error reading file")
    }
    
    return loadedCommands
}

func checkForConsensus(numberOfNodes int, time_of_consensus int64, time_before_gossip int64, time_after_gossip int64, resultsFile *os.File) {
	fmt.Println("Checking for consensus...")
	if atomic.LoadInt32(gossiping) == 0 && atomic.LoadInt32(updateBackLog) == 0 {
		fmt.Println("Not running")
		return
	} 
	consensus_bool := false
	previous := int32(0)
	previousConsensus := int32(0)
    for (atomic.LoadInt32(gossiping) > 0 && atomic.LoadInt32(updateBackLog) > 0) || consensus_bool == false {
    	if atomic.LoadInt32(consensus) == int32(numberOfNodes) && consensus_bool == false {
    		time_of_consensus = time.Now().UnixNano()
    		consensus_bool = true
    	}
    	current := atomic.LoadInt32(gossiping)
    	currentConsensus := atomic.LoadInt32(consensus)
    	if current != previous {
    		/*fmt.Println("Gossiping is:", atomic.LoadInt32(gossiping))
			fmt.Println("updateBackLog is:", atomic.LoadInt32(updateBackLog))
    		fmt.Println("Nodes gossiping is:", current)*/
    	}
    	
    	if currentConsensus != previousConsensus {
    		fmt.Println("Consensus progress is:", atomic.LoadInt32(consensus))
    	}
    	
    	previous = current
    	previousConsensus = currentConsensus
    	if (time.Now().UnixNano()-time_before_gossip)/int64(time.Millisecond) > 61000 {
    		fmt.Println("Been over a minute, exiting...")
    		break
    	}
    }
    time_after_gossip = time.Now().UnixNano()
    if consensus_bool == true {
    	fmt.Println("Time for consensus in Milliseconds:",(time_of_consensus-time_before_gossip)/int64(time.Millisecond))
    	fmt.Fprintln(resultsFile, "Time for consensus in Milliseconds:",(time_of_consensus-time_before_gossip)/int64(time.Millisecond))
    } else {
    	fmt.Println("Time for consensus: Not Reached")
    	fmt.Fprintln(resultsFile, "Time for consensus: Not Reached")
    }
    fmt.Println("Time for gossip to end in Milliseconds:",(time_after_gossip-time_before_gossip)/int64(time.Millisecond))
    fmt.Fprintln(resultsFile, "Time for gossip to end in Milliseconds:",(time_after_gossip-time_before_gossip)/int64(time.Millisecond))
    atomic.AddInt32(consensus, -atomic.LoadInt32(consensus))
    fmt.Println("[END]")
    fmt.Fprintln(resultsFile, "[END]")
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	debug.SetGCPercent(-1)
	
	resultsFile, resultsFileErr := os.Create("results.txt")
	if resultsFileErr != nil {
	    fmt.Println(resultsFileErr)
	    return
	}

	neighbourListSizePercentage := float64(5)
	version := 0
	
	time_before_gossip := int64(0)
	time_of_consensus := int64(0)
	time_after_gossip := int64(0)
	
	loadedCommands := readFromFile()
	atomic.AddInt32(randomUpdateInterval, 60)
	//currentNetworkSize := 0
		
	var nodes []node
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Starting blockchain simulation program")
	fmt.Fprintln(resultsFile, "Starting blockchain simulation program")
	
    for true {
    	currCommand := ""
    	if len(loadedCommands) > 0 {
    		currCommand = loadedCommands[0]
    		loadedCommands[0] = ""
    		loadedCommands = loadedCommands[1:]    		
    	} else {
    		scanner.Scan()
	    	currCommand = scanner.Text()
    	}
    	currCommandElements := strings.Fields(currCommand)
    	if (len(currCommandElements) == 0) {
    		continue
    	}
    	if currCommandElements[0] == "SPAWN" && len(currCommandElements) == 3 {
    		if spawnAmount, err := strconv.Atoi(currCommandElements[1]); err == nil {
    			if currCommandElements[2] != "PUSH" || currCommandElements[2] != "PUSH&PULL" {
    				nodeType := currCommandElements[2]
    				if spawnAmount > 0 {
				    	if len(nodes) > 0 {
				    		killRoutines(&nodes)
				    		version = 0
				    	}
				    	neighbourListSize := int(neighbourListSizePercentage*float64(spawnAmount)/float64(100))
				    	if neighbourListSize < 1 {
				    		neighbourListSize = 1
				    	}
				    	fmt.Println("Neighbour List Size Percentage Is:", neighbourListSize)
				    	spawnFunction(spawnAmount, &nodes, neighbourListSize, nodeType)
				    	fmt.Println("[COMPLETE] SPAWN")
				    	fmt.Println("Number of goroutines is:", runtime.NumGoroutine())
		    		}
    			}
    		}
    	} else if currCommandElements[0] == "BROADCAST" && len(currCommandElements) == 1 {
    		if len(nodes) > 0 {
    			
    			//logic to broadcast
    		}
    	} else if currCommandElements[0] == "KILL" && len(currCommandElements) == 1 {
    		if len(nodes) > 0 {
    			killRoutines(&nodes)
    			version = 0
    			fmt.Println("[COMPLETE] KILL")
    		}
    	} else if currCommandElements[0] == "STATUS" && len(currCommandElements) == 1 {
    		if len(nodes) > 0 {
    			var networkStatusData []networkStatus
    			totalMessages := 0
    			for i := 0; i < len(nodes); i++ {
    				totalMessages = totalMessages + nodes[i].numberOfUpdates
    				found := false
    				for j := 0; j < len(networkStatusData); j++ {
    					if nodes[i].version == networkStatusData[j].version {
    						networkStatusData[j].count++
    						found = true
    						break;
    					}
    				}
    				if found == false {
    					var newNetworkStatusDataEntry networkStatus
    					newNetworkStatusDataEntry.version = nodes[i].version
    					newNetworkStatusDataEntry.count = 1
    					networkStatusData = append(networkStatusData, newNetworkStatusDataEntry)
    				}
    			}
    			
    			for i := 0; i < len(networkStatusData); i++ {
    				fmt.Println("Version:", networkStatusData[i].version, "Count:", networkStatusData[i].count)
    				fmt.Fprintln(resultsFile, "Version:", networkStatusData[i].version, "Count:", networkStatusData[i].count)
    			}
    			fmt.Println("Total Messages Sent Is:", totalMessages)
    			fmt.Fprintln(resultsFile, "Total Messages Sent Is:", totalMessages)
    			fmt.Println("[COMPLETE] STATUS")
    		}
    	} else if currCommandElements[0] == "UNICAST" && len(currCommandElements) == 2 {
    		if nodeIndex, err := strconv.Atoi(currCommandElements[1]); err == nil {
			    if nodeIndex >= 0 && nodeIndex < len(nodes) {
			    	numberOfNodes := len(nodes)
			    	neighbourListSize := int(neighbourListSizePercentage*float64(len(nodes))/float64(100))
			    	fmt.Println("[START]", neighbourListSizePercentage, "%")
			    	fmt.Fprintln(resultsFile, "[START]", neighbourListSizePercentage, "%")
			    	//fmt.Println("Beginning Gossip for:", len(nodes), "nodes")
			    	fmt.Fprintln(resultsFile, "Beginning Gossip for:", len(nodes), "nodes")
			    	//fmt.Println("NeighbourList Percentage:", neighbourListSizePercentage, "%")
			    	fmt.Fprintln(resultsFile, "NeighbourList Percentage:", neighbourListSizePercentage, "%")
			    	//fmt.Println("NeighbourList Size:", neighbourListSize)
			    	fmt.Fprintln(resultsFile, "NeighbourList Size:", neighbourListSize)
			    	version++
			    	time_before_gossip = time.Now().UnixNano()
			    	atomic.AddInt32(updateBackLog, 1)
			    	nodes[nodeIndex].channel <- version
	    			for atomic.LoadInt32(gossiping) <= 0 && atomic.LoadInt32(updateBackLog) <= 0 {
	    				//fmt.Println("Waiting for gossip to start")
			    	}
	    			checkForConsensus(numberOfNodes, time_of_consensus, time_before_gossip, time_after_gossip, resultsFile)
	    			fmt.Println("[COMPLETE] UNICAST")
	    		}
    		}
    	} else if currCommandElements[0] == "ROUTINES" && len(currCommandElements) == 1 {
    		fmt.Println("Current number of routines running is:", runtime.NumGoroutine())
    		fmt.Println("[COMPLETE] ROUTINES")
    	} else if currCommandElements[0] == "UPDATENLISTSIZE" && len(currCommandElements) == 2 {
    		if newSize, err := strconv.Atoi(currCommandElements[1]); err == nil {
    			fmt.Println("[BEGIN] UPDATENLISTSIZE", newSize)
    			if newSize > 100 {
    				neighbourListSizePercentage = float64(100)
    			} else if newSize < 1 {
    				neighbourListSizePercentage = float64(1)
    			} else {
    				neighbourListSizePercentage = float64(newSize)
    			}
    			fmt.Println("[END] UPDATENLISTSIZE")
    		} else {
    			fmt.Println("[ERROR] UPDATENLISTSIZE", newSize, "- Use An Integer As Second Argument")
    		}
    	} else if currCommandElements[0] == "RESET" && len(currCommandElements) == 1 {
    		if len(nodes) > 0 {
    			fmt.Println("[BEGIN] RESET")
    			version = 0
    			atomic.AddInt32(gossiping, -atomic.LoadInt32(gossiping))
    			atomic.AddInt32(consensus, -atomic.LoadInt32(consensus))
    			atomic.AddInt32(updateBackLog, -atomic.LoadInt32(updateBackLog))
    			runtime.GC()
    			debug.FreeOSMemory()
    			fmt.Println("[END] RESET")
    		} else {
    			fmt.Println("[ERROR] RESET - No Nodes Spawned")
    		}
    	} else if currCommandElements[0] == "UPDATEINTERVALTIME" && len(currCommandElements) == 2 {
    		if newInterval, err := strconv.Atoi(currCommandElements[1]); err == nil {
    			fmt.Println("[BEGIN] UPDATEINTERVALTIME", newInterval)
    			atomic.AddInt32(randomUpdateInterval, int32(newInterval)-atomic.LoadInt32(randomUpdateInterval))
    			fmt.Println("[END] UPDATEINTERVALTIME", newInterval)
    		} else {
    			fmt.Println("[ERROR] UPDATEINTERVALTIME", newInterval, "- Use An Integer As Second Argument")
    		}
    	}
    }
    
    if scanner.Err() != nil {
        fmt.Println(os.Stderr, "reading standard input:", scanner.Err())
    }
}


