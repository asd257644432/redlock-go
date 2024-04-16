package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asd257644432/redlock-go/v3/redlock"
)

const (
	fpath = "./counter.log"
)

func writer(count int, back chan string) {
	ctx := context.Background()
	lock, err := redlock.NewRedLock(
		ctx,
		[]string{
			"tcp://127.0.0.1:6379",
			"tcp://127.0.0.1:6380",
			"tcp://127.0.0.1:6381",
		},
		redlock.WithCacheType(redlock.CacheTypeFreeCache),
		redlock.WithCacheSize(20*1024*1024),
	)

	if err != nil {
		panic(err)
	}

	incr := 0
	for i := 0; i < count; i++ {
		expiry, err := lock.Lock(ctx, "foo", 1000*time.Millisecond)
		if err != nil {
			fmt.Println(err)
		} else {
			if expiry > 500 {
				f, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE, os.ModePerm)
				if err != nil {
					panic(err)
				}
				defer f.Close()
				buf := make([]byte, 1024)
				n, _ := f.Read(buf)
				num, _ := strconv.ParseInt(strings.TrimRight(string(buf[:n]), "\n"), 10, 64)
				f.WriteAt([]byte(strconv.Itoa(int(num+1))), 0)
				incr += 1
				lock.UnLock(ctx, "foo")
			}
		}
	}
	fmt.Printf("%s increased %d times.\n", fpath, incr)
	back <- "done"
}

func init() {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(err)
	}
	f.WriteString("0")
	defer f.Close()
}

func main() {
	threads := 5
	inc := 100
	done := make(chan string, threads)
	for i := 0; i < threads; i++ {
		go writer(inc, done)
	}
	for i := 0; i < threads; i++ {
		<-done
	}

	f, err := os.OpenFile(fpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	buf := make([]byte, 1024)
	n, _ := f.Read(buf)
	fmt.Printf("Counter value is %s\n", buf[:n])
}
