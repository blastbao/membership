package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {

	portPtr := flag.Int("p", 10000, "Port the process will be listening on for incoming messages")
	pausePtr := flag.Int("pause", 0, "Sleep for specified time after startup")
	hostfilePtr := flag.String("h", "hostfile", "Path to hostfile")

	flag.Parse()

	// Sleep before continuing. Use to delay start of follower
	time.Sleep(time.Duration(*pausePtr) * time.Second)

	// read host file
	hosts := getHostListFromFile(*hostfilePtr)

	hostname, err := os.Hostname()
	LogFatalCheck(err, "Error retrieving hostname")

	ipHostMap := make(map[string]string)
	pidMap := make(map[string]int)

	for i, h := range hosts {
		pidMap[h] = i
		ipAddrs, _ := net.LookupHost(h)
		ipHostMap[ipAddrs[0]] = h
	}

	if hosts[0] == hostname {
		log.Printf("Master: %s", hostname)
		StartLeader(ipHostMap, pidMap, *portPtr)
	} else {
		StartFollower(ipHostMap, hosts[0], *portPtr)
	}
}

func getHostListFromFile(filePath string) []string {
	f, err := os.Open(filePath)
	LogFatalCheck(err, fmt.Sprintf("Cannot read hostfile at %s", filePath))
	defer f.Close()

	scanner := bufio.NewScanner(f)
	LogFatalCheck(scanner.Err(), "Error scanning file")

	var s []string

	for scanner.Scan() {
		s = append(s, scanner.Text())
	}

	return s
}
