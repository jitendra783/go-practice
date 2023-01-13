package random

import "log"

func Mapmap(){
	mpmp := map[int]map[string]string{
		1: {
            "a": "Apple",
            "b": "Banana",
            "c": "Coconut",
        },
        2: {
            "a": "Tea",
            "b": "Coffee",
            "c": "Milk",
        },
        3: {
            "a": "Italian Food",
            "b": "Indian Food",
            "c": "Chinese Food",
        },
	}

	log.Println("len:", len(mpmp))
	
	for k , v := range mpmp{
		log.Println(k , v)
	}
	delete(mpmp, 1)
    log.Println("map:", mpmp)

}