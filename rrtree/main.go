// package main

// import (
// 	"fmt"

// 	"github.com/miekg/dns"
// )

// func main() {
// 	node, partial := DefaultTree.GetNode("c.root-servers.net")
// 	fmt.Println(partial)
// 	if rrs, ok := node.Resources[dns.TypeA]; ok {
// 		fmt.Printf("%+v\n", rrs)
// 		fmt.Printf("%+v\n", rrs[0])
// 	}

// 	return
// }
