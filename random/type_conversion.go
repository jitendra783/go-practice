package random
import (
	"log"
    "reflect"
)
func Type_Conversion(){
	var a,d,e int 
	var b,f float64
	var c,g string 
    a =6 
	b =2.99
	c = "njgfdkj"
    d= float64(a) + b
	log.Println(a,b,c,d)
	f=float64(a)+b
	log.Println(a,b,c,d,f)

	e= a+int(f)
	log.Println(f,e)
	g = c 
	log.Println(a,c)
	d= float64(a)+ b 
	log.Println(a,d)
     
}
