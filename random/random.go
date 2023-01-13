package random

import (
	"log"
	"sync"
	"fmt"
	"time"
)

func RandomSleep(wg *sync.WaitGroup, s string, l ,t int) {
	log.Println("")
	defer wg.Done()
    for i:= 1; i<= l ;i++{
        fmt.Println(s," ", i)
        time.Sleep(time.Duration(t * int(time.Second))) 
    }
}
func RandomSleep1(wg *sync.WaitGroup, s string, l ,t int) {
	defer wg.Done()
    for i:= 1; i<= l ;i++{
        fmt.Println(s, " ",i)
        time.Sleep(time.Duration(t * int(time.Second))) 
    }

	log.Println("")

}
func RandomSleep2(wg *sync.WaitGroup, s string, l ,t int) {
	defer wg.Done()
    for i:= 1; i<= l ;i++{
        fmt.Println(s," ", i)
        time.Sleep(time.Duration(t * int(time.Second))) 
    }

	log.Println("")

}
func RandomSleep3(wg *sync.WaitGroup, s string, l ,t int) {
	defer wg.Done()
    for i:= 1; i<= l ;i++{
        fmt.Println(s," ", i)
        time.Sleep(time.Duration(t * int(time.Second))) 
    }
	log.Println("")

}