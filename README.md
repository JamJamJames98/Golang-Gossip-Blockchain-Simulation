# How To Run
Run the 'main.go' file as a normal golang file (i.e. will the following command in a shell)

*go run main.go*

# Doc
Avaiable commands in the suite:

*SPAWN <NUMBER> <NODE_TYPE>*
* Recommended number between 100 and 1000
* Node types available:
  * PUSH
  * PULL
  * PUSH&PULL
 
*KILL*
* Kills all nodes in the current network

*UNICAST <NODE_INDEX>*
* Sends an update to the node index as specified, begins the gossip

*STATUS*
* Returns network status

*BROADCAST*
* Sends an update to all nodes in the network

*UPDATENLISTTYPE <TYPE>*
* Neighbour list types available:
  * Flat
  * Percent
  
*UPDATENLISTPERCENT <NUMBER>*
* Update the percentage of the network to be on the neighbour list
* Takes in a number from 1 to 100
 
*UPDATENLISTSIZE <NUMBER>*
* Update the number of neighbours to be on the neighbour list
* Takes in a number
 
*UPDATEINTERVALTIME <NUMBER>*
* Update the time range PUSH nodes wait before requesting for updates
* Takes in a number
 
*RESET*
* Resets the network
