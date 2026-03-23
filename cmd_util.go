package main

import (
	"fmt"
	"net"
	"strconv"
)

func cmdFreeport() error {
	port, err := freePort()
	if err != nil {
		return err
	}
	fmt.Println(port)
	return nil
}

// execFreeport is the test-script version of freeport.
// Syntax: freeport PORT   → stores port in ${PORT}
// Syntax: freeport        → stores port in ${result}
func execFreeport(args []string, vars map[string]string) error {
	port, err := freePort()
	if err != nil {
		return err
	}

	varName := "result"
	if len(args) > 0 {
		varName = args[0]
	}

	vars[varName] = strconv.Itoa(port)
	return nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to find free port: %w", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}
