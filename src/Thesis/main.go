package main

import (
	"fmt"
	"sync"
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
 *	Orchestrator method to kill all waiting threads, calls killNode
 */
func killRoutines(nodes *[]node) {
	for i := 0; i < len(*nodes); i++ {
		(*nodes)[i].channel <- -1
	}
	for runtime.NumGoroutine() > 1 {
		time.Sleep(1)
		//wait for threads to die 
	}
	(*nodes) = nil
	atomic.AddInt32(gossiping, -atomic.LoadInt32(gossiping))
    atomic.AddInt32(consensus, -atomic.LoadInt32(consensus))
    atomic.AddInt32(updateBackLog, -atomic.LoadInt32(updateBackLog))
    //fmt.Println("Gossiping is:", atomic.LoadInt32(gossiping))
    runtime.GC()
    debug.FreeOSMemory()
}

/*
 *	Runnable node method
 */
func runNode(id int, name string, myNode *node, fanOut int, size int) {
	runtime.Gosched()
	//ensure that all spawning is complete
	for runtime.NumGoroutine() < size {
		time.Sleep(1)
		//wait for all nodes to spawn
	}

	for {
		randTime := rand.Intn(60)
		afterChannel := time.After(time.Duration(randTime) * time.Second)
		//fmt.Println("Routines:", runtime.NumGoroutine())
		select {
		case recievedValue := <- (*myNode).channel:
			atomic.AddInt32(gossiping, 1)
			if recievedValue == -1 {
				atomic.AddInt32(gossiping, -1)
				return
			} else if recievedValue > 0 {	
				sleepDuration := time.Duration(rand.Intn(600 - 40 + 1) + 40) * time.Millisecond
				time.Sleep(sleepDuration)
				if recievedValue >= (*myNode).version {	
					//update here
					if recievedValue > (*myNode).version {
						atomic.AddInt32(consensus, 1)
						(*myNode).version = recievedValue
						//fmt.Println("Setting node", (*myNode).id, "to version", recievedValue)
						//gossip to all neighbours
						for i := 0; i < len((*myNode).neighbourNodes); i++ {
							//check edge case if neighbour is set to itself (random neighbour list generation sometimes causes this)
							if (*myNode).neighbourNodes[i].channel != (*myNode).channel {
								atomic.AddInt32(updateBackLog, 1)
								//fmt.Println("Sending:", recievedValue, "To overwrite:", (*myNode).neighbourNodes[i].version, "At Node id:'", (*myNode).neighbourNodes[i].id, "' From Node id:'", (*myNode).id, "'")
								(*myNode).neighbourNodes[i].channel <- (*myNode).version
								(*myNode).numberOfUpdates++
							}
						}
						//fmt.Println("Finished sending to neighbours")
					} else {
						//fmt.Println("[DUPLICATE] - Node ID:", (*myNode).id)
					}
				} else {	
					//do nothing with old or malformed version
				}
			}
			//fmt.Println("Decrementing gossip for node at id:", id)
			atomic.AddInt32(updateBackLog, -1)
			atomic.AddInt32(gossiping, -1)
		case <- afterChannel:
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
 *	Orchestrator method to spawn waiting threads, calls runNode
 */
func spawnFunction(size int, nodes *[]node, neighbourListSize int, fanOut int) {
	(*nodes) = make([]node, size)
	//prev := float64(0)
	//fmt.Println("Neighbour list size is:", neighbourListSize)
	//fmt.Println("Initialising...")
	for i := 0; i < size; i++ {
		//prev = progressUpdate(i, size, prev, "Spawn Progress: ")
		(*nodes)[i].channel = make(chan int, size+1)
		(*nodes)[i].requestChannel = make(chan RequestData, size+1)
		(*nodes)[i].requestData.channel = (*nodes)[i].channel
		(*nodes)[i].id = i
		(*nodes)[i].version = 0
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
		go runNode(i, "Spawning", &(*nodes)[i], fanOut, size)
	}
	//fmt.Println("Spawn Function Finished")
}

func readFromFile() []string {
	loadedCommands := make([]string, 0)
	//for running from eclipse
	//fileToRead, error := os.Open("commands.txt")
	//for running from terminal
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
	//fmt.Println("Gossiping is:", atomic.LoadInt32(gossiping))
	//fmt.Println("updateBackLog is:", atomic.LoadInt32(updateBackLog))
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
    		//fmt.Println("Gossiping is:", atomic.LoadInt32(gossiping))
			//fmt.Println("updateBackLog is:", atomic.LoadInt32(updateBackLog))
    		//fmt.Println("Nodes gossiping is:", current)
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
    //fmt.Println("Nodes gossiping is:", atomic.LoadInt32(gossiping))
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
	//TODO: remove len(nodes) from everywhere - have somewhat global variable and access that
	rand.Seed(time.Now().UTC().UnixNano())
	debug.SetGCPercent(-1)
	
	resultsFile, resultsFileErr := os.Create("results.txt")
	if resultsFileErr != nil {
	    fmt.Println(resultsFileErr)
	    return
	}
	/*Variables in the suite:
	Ping
	*/
	
	/*Parameters in parameter based gossip:
	UNSURE:Messaging Strategies / Push & Pull
	*/
	fanOut := 4
	//timeBetweenRounds := 10
	neighbourListSizePercentage := float64(5)
	//fmt.Println(neighbourListSizePercentage)
	version := 0
	
	time_before_gossip := int64(0)
	time_of_consensus := int64(0)
	time_after_gossip := int64(0)
	
	loadedCommands := readFromFile()
		
	var wg sync.WaitGroup
	var nodes []node
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Starting blockchain simulation program")
	fmt.Fprintln(resultsFile, "Starting blockchain simulation program")
	
    for true {
    	wg.Wait()
    	//fmt.Println("Routines running:",runtime.NumGoroutine())
    	//if statement to use loadedCommands instead of user input
    	currCommand := ""
    	if len(loadedCommands) > 0 {
    		currCommand = loadedCommands[0]
    		//clear and remove this entry from the loaded commands
    		loadedCommands[0] = ""
    		loadedCommands = loadedCommands[1:]    		
    	} else {
    		scanner.Scan()
	    	currCommand = scanner.Text()
    	}
    	//time.Sleep(1)
    	currCommandElements := strings.Fields(currCommand)
    	if (len(currCommandElements) == 0) {
    		continue
    	}
    	if currCommandElements[0] == "SPAWN" && len(currCommandElements) == 2 {
    		if spawnAmount, err := strconv.Atoi(currCommandElements[1]); err == nil {
			    if spawnAmount > 0 {
			    	if len(nodes) > 0 {
			    		killRoutines(&nodes)
			    		version = 0
			    	}
			    	//fmt.Println("Threads running before new spawn is:", runtime.NumGoroutine())
			    	//fmt.Println("Spawning",spawnAmount,"nodes")
			    	neighbourListSize := int(neighbourListSizePercentage*float64(spawnAmount)/float64(100))
			    	if neighbourListSize < 1 {
			    		neighbourListSize = 1
			    	}
			    	fmt.Println("Neighbour List Size Percentage Is:", neighbourListSize)
			    	spawnFunction(spawnAmount, &nodes, neighbourListSize, fanOut)
			    	fmt.Println("[COMPLETE] SPAWN")
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
    			if newSize > 100 {
    				neighbourListSizePercentage = float64(100)
    			} else if newSize < 1 {
    				neighbourListSizePercentage = float64(1)
    			} else {
    				neighbourListSizePercentage = float64(newSize)
    			}
    			neighbourListSize := int(neighbourListSizePercentage*float64(len(nodes))/float64(100))
			    if neighbourListSize < 1 {
			    	neighbourListSize = 1
			    }
			    
			    spawnAmount := len(nodes)
			    
			    if len(nodes) > 0 {
			    	killRoutines(&nodes)
			    	version = 0
			    }
			    fmt.Println("Neighbour List Size Percentage Is:", neighbourListSize)
			    spawnFunction(spawnAmount, &nodes, neighbourListSize, fanOut)
    			fmt.Println("[COMPLETE] UPDATENLISTSIZE")
    		}
    	} else if currCommandElements[0] == "RESET" && len(currCommandElements) == 1 {
    		if len(nodes) > 0 {
    			version = 0
    			atomic.AddInt32(gossiping, -atomic.LoadInt32(gossiping))
    			atomic.AddInt32(consensus, -atomic.LoadInt32(consensus))
    			atomic.AddInt32(updateBackLog, -atomic.LoadInt32(updateBackLog))
    			//fmt.Println("Gossiping is:", atomic.LoadInt32(gossiping))
    			runtime.GC()
    			debug.FreeOSMemory()
    			fmt.Println("[COMPLETE] RESET")
    		}
    	}
    }
    
    if scanner.Err() != nil {
        fmt.Println(os.Stderr, "reading standard input:", scanner.Err())
    }
}


