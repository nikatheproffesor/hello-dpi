package main

/*
#include <stdint.h>
#include <stdbool.h>
*/
import "C"
import (
	"github.com/hellodpi/hellodpi/pkg/hellocore"
)

//export HelloCoreStart
func HelloCoreStart(port C.int) C.int {
	p := int(port)
	if p <= 0 {
		p = 18080
	}
	err := hellocore.StartMobileDefault(p)
	if err != nil {
		return -1
	}
	return 0
}

//export HelloCoreStop
func HelloCoreStop() C.int {
	err := hellocore.StopMobileDefault()
	if err != nil {
		return -1
	}
	return 0
}

//export HelloCoreIsRunning
func HelloCoreIsRunning() C.bool {
	return C.bool(hellocore.IsMobileRunning())
}

func main() {}
