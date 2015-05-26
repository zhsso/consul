package main

import (
	"fmt"
	"os"

	"time"

	consulApi "github.com/hashicorp/consul/api"
	"github.com/zhsso/consul"
)

func aaa(a []*consulApi.ServiceEntry) {
	fmt.Println(time.Now().Unix())
	for k, v := range a {
		fmt.Printf("\t\n", k)
		fmt.Println()
	}
}

func main() {
	s, err := consul.NewServiceAgent("redis", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	s.SetCallBack(aaa)
	s.RegisterService(os.Args[1], "127.0.0.1", 8987, "1232")
	for {
		time.Sleep(time.Second * 100)
	}
}
