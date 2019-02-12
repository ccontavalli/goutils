package misc

import (
	//"github.com/stretchr/testify/assert"
	//"testing"
	"fmt"
)

func ExampleNaiveDir() {
	fmt.Println(NaiveDir("/a/b/c"))
	fmt.Println(NaiveDir("a/b/c"))
	fmt.Println(NaiveDir("/a/"))
	fmt.Println(NaiveDir("a/"))
	fmt.Println(NaiveDir("/"))
	fmt.Println(NaiveDir(""))

	// Output:
	// /a/b
	// a/b
	// /a
	// a
	// /
	//
}
