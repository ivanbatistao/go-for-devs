package main

import (
	"flag"
	"fmt"
)

func greeting(name string, age int) string {
	return fmt.Sprintf("Hello, %s. I see your age is %d\n", name, age)

}

func main() {
	var name string

	flag.StringVar(&name, "name", "", "name of the user")
	agePtr := flag.Int("age", 0, "age of the user")

	flag.Parse()

	fmt.Println(greeting(name, *agePtr))
}
