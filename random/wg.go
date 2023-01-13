package random

import (
	"sync"
)

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	RandomSleep(wg, "string1 ", 5, 2)
	RandomSleep(wg, "string2 ", 4, 3)

	go RandomSleep1(wg, "stringf1 ", 5, 3)
	RandomSleep1(wg, "stringf2 ", 4, 3)

	RandomSleep2(wg, "stringf1 ", 5, 3)
	go RandomSleep2(wg, "stringf1 ", 4, 3)

	go RandomSleep3(wg, "stringf1 ", 5, 3)
	go RandomSleep3(wg, "stringf2 ", 4, 3)

	wg.Wait()
}
