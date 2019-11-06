import sys
import os

#setup results object
class Results(object):
	totalNodes = -1
	neighbourListSize = -1
	consensusHit = 0
	consensusMiss = 0
	avgConsensusTime = 0
	avgGossipTime = 0
	avgNodesNotReached = 0
	avgNodesReached = 0
	avgtotalMessagesSent = 0
	quantity = 0

#define method
def processResults(result):
	if (result.totalNodes == -1):
		return

	if (result.avgConsensusTime != 0):
		result.avgConsensusTime = result.avgConsensusTime/result.consensusHit

	result.consensusMiss = result.consensusMiss/result.quantity
	result.consensusHit = result.consensusHit/result.quantity
	result.avgGossipTime = result.avgGossipTime/result.quantity
	result.avgNodesReached = result.avgNodesReached/result.quantity
	result.avgNodesNotReached = result.avgNodesNotReached/result.quantity
	result.avgtotalMessagesSent = result.avgtotalMessagesSent/result.quantity

def printResultsAttributes(result, neighbourListPercentage):
	if (result.totalNodes == -1):
		return

	print("For Node With: ", neighbourListPercentage, " Neighbours")
	print("Total Nodes: ", result.totalNodes)
	print("Neighbour List Size: ", result.neighbourListSize)
	print("Consensus Miss: ", result.consensusMiss)
	print("Consensus Hit: ", result.consensusHit)
	print("Average Consensus Time: ", result.avgConsensusTime)
	print("Average Gossip Time: ", result.avgGossipTime)
	print("Average Nodes Reached: ", result.avgNodesReached)
	print("Average Nodes Not Reached: ", result.avgNodesNotReached)
	print("Average Total Messages Sent ", result.avgtotalMessagesSent)
	print("Quantity Of Results: ", result.quantity)

def getNewLine(file, first_line):
	line = file.readline().rstrip()
	if not line and first_line == True:
		return -1
	if not line and first_line == False:
		print("Should be a line here, something went wrong")
		exit(1)
	if (line == "[END]"):
		return getNewLine(file, first_line)
	return line

def getIndex(file):
	line = getNewLine(file, True)
	line = getNewLine(file, True)
	if (line == -1):
		return -1
	return int(line.partition("Beginning Gossip for: ")[2].partition(" ")[0])

def getNeighbourListSize(file):
	line = getNewLine(file, False)
	return int(line.partition("NeighbourList Size: ")[2])

def getConsensus(file):
	line = getNewLine(file, False)
	if (line.partition("Time for consensus: ")[2] == "Not Reached"):
		return False
	else:
		if (line.partition("Time for consensus in Milliseconds: ")[2] != ""):
			return int(line.partition("Time for consensus in Milliseconds: ")[2])
		else:
			return int(line.partition("Time for consensus: ")[2])

def getGossiping(file):
	line = getNewLine(file, False)
	return int(line.partition("Time for gossip to end in Milliseconds: ")[2])

def getNodesReached(file):
	line = getNewLine(file, False)
	if (line.partition("Version: 1 Count: ")[2] == ""):
		return 0
	return int(line.partition("Version: 1 Count: ")[2])

def getNodesNotReached(file):
	line = getNewLine(file, False)
	print(line)
	return int(line.partition("Version: 0 Count: ")[2])

def getMessagesSent(file):
	line = getNewLine(file, False)
	return int(line.partition("Total Messages Sent Is: ")[2])

#begin main logic
if (len(sys.argv) != 2):
	print("Incorrect usage, please pass along the results file name as a parameter")
	exit(0)

fileReadable = os.access(sys.argv[1], os.R_OK)
if (fileReadable == False):
	print("Please make sure your results file is in the current directory and is readable")
	exit(0)

fileName = sys.argv[1]

resultsList = [Results() for i in range(6000)]

file = open(fileName)
line = file.readline()
counter = 0

while True:
	index = getIndex(file)

	if (index == -1):
		break

	counter += 1
	print(counter)

	resultsList[index].quantity += 1
	#print(resultsList[index].quantity)
	resultsList[index].totalNodes = index
	#print(resultsList[index].totalNodes)

	if (resultsList[index].neighbourListSize == -1):
		resultsList[index].neighbourListSize = getNeighbourListSize(file)
	else:
		if resultsList[index].neighbourListSize != getNeighbourListSize(file):
			print("Incompatible results file, use the same node size setup throughout")
			exit(1)
	#print(resultsList[index].neighbourListSize)
	consensus = getConsensus(file)
	if (consensus == False):
		resultsList[index].consensusMiss += 1
	else:
		resultsList[index].consensusHit += 1
		resultsList[index].avgConsensusTime += consensus
	#print(resultsList[index].consensusHit)
	#print(resultsList[index].consensusMiss)
	#print(resultsList[index].avgConsensusTime)

	resultsList[index].avgGossipTime += getGossiping(file)
	#print(resultsList[index].avgGossipTime)

	nodesReached = getNodesReached(file)
	resultsList[index].avgNodesReached += nodesReached
	if (nodesReached == 0):
		resultsList[index].avgNodesNotReached += resultsList[index].totalNodes
	elif (nodesReached == resultsList[index].totalNodes):
		resultsList[index].avgNodesNotReached += 0
	else:
		resultsList[index].avgNodesNotReached += getNodesNotReached(file)

	resultsList[index].avgtotalMessagesSent += getMessagesSent(file)
	#print(resultsList[index].avgtotalMessagesSent)

file.close()
print("Processed", counter, "results")

for x in range(len(resultsList)):
	if (x == 0):
		continue
	processResults(resultsList[x])
	printResultsAttributes(resultsList[x], x)
