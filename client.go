package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var host = flag.String("host", "140.116.54.31", "host")
var port = flag.String("port", "4001", "port")

func main() {
	flag.Parse()
	fmt.Println("Connect to : " + *host + ":" + *port)

	conn, err := net.DialTimeout("tcp", *host+":"+*port, 3*time.Second)

	if err != nil {
		fmt.Println("Error Connecting", err)
		os.Exit(1)
	}

	defer conn.Close()

	fmt.Println("Connecting to : " + *host + ":" + *port + ", status: OK")

	var wg sync.WaitGroup
	wg.Add(2)

	//go handleWrite(sendtext, conn, &wg)
	go handleRead(conn, &wg)

	wg.Wait()

}

func handleWrite(sendtext string, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	_, e := conn.Write([]byte(sendtext))

	if e != nil {
		fmt.Println("Error to send message because of", e.Error())
	}
}

func handleRead(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()
	var byteArr []byte
	var arr []string
	var hexlen string
	var SEQ byte
	var data uint16
	i := 0

	for {

		tmp := make([]byte, 1)

		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}

		//fmt.Println("total bytes:", n)

		if n > 0 {

			byteArr = append(byteArr, tmp[0])

			//fmt.Printf("byte result: %v \n", tmp[0])

			if i == 2 {
				for _, s := range tmp {
					SEQ = s
				}
			}

			a := hex.EncodeToString(tmp[:n])
			arr = append(arr, a)

			//fmt.Printf("hex result: %s \n", a)

			if i == 5 {
				hexlen = a
			}

			if i == 6 {
				hexlen += a
				m, err := hex.DecodeString(hexlen)
				if err != nil {
					fmt.Println(err)
				}

				data = binary.BigEndian.Uint16(m)

				//fmt.Println("real length:", data)

			}

			i = i + 1

			//fmt.Println("i:", i)

			if i == int(data) {
				fmt.Printf("byte slice result: %v \n", byteArr)
				fmt.Printf("hex slice result: %v \n", arr)

				//檢查CKS是否正確
				var firstbit byte
				var xor byte
				for j, b := range byteArr {
					if j == 0 {
						firstbit = b
					}

					if j == 1 {
						xor = firstbit ^ b
					}

					if j > 1 && j < (len(byteArr)-1) {
						xor = xor ^ b
					}

					if j == (len(byteArr) - 1) {
						hexXOR := strconv.FormatInt(int64(xor), 16)
						CKS := strconv.FormatInt(int64(b), 16)
						if hexXOR == CKS {
							fmt.Printf("CKS [%v] MATCH hexXOR [%v] \n", CKS, hexXOR)
						} else {
							fmt.Println("CKS NO MATCH hexXOR")
						}
					}
				}

				//檢查長度是否正確
				if string(len(arr)) == string(data) {
					fmt.Println("Length Match")
				} else {
					fmt.Println("Length No Match")
				}

				//回覆ACK
				var PACK []byte
				var AckCks byte
				var DLE byte
				var ACK byte
				var ADDR1 byte
				var ADDR2 byte
				var LenA byte
				var LenB byte

				for x := 1; x <= 8; x++ {

					if x == 1 {
						DLE = byte(0xAA)
						PACK = append(PACK, DLE)
					}

					if x == 2 {
						ACK = byte(0xDD)
						PACK = append(PACK, ACK)
					}

					if x == 3 {
						PACK = append(PACK, SEQ)
					}

					if x == 4 {
						ADDR1 = byte(0x34)
						PACK = append(PACK, ADDR1)
					}

					if x == 5 {
						ADDR2 = byte(0x63)
						PACK = append(PACK, ADDR2)
					}

					if x == 6 {
						LenA = byte(0x00)
						PACK = append(PACK, LenA)
					}

					if x == 7 {
						LenB = byte(0x08)
						PACK = append(PACK, LenB)
					}

					if x == 8 {
						AckCks = DLE ^ ACK ^ SEQ ^ ADDR2 ^ ADDR1 ^ LenB ^ LenA
						fmt.Printf("AckCks: %v \n", AckCks)
						PACK = append(PACK, AckCks)
						fmt.Printf("PACK %v \n", PACK)

						_, e := conn.Write(PACK)

						if e != nil {
							fmt.Println("Error to send message because of", e.Error())
						}

						arr = nil
						byteArr = nil
						PACK = nil
						i = 0

					}
				}

				elapsed := time.Since(start)
				fmt.Printf("Response Time is %d ms \n", elapsed.Nanoseconds()/10000000)
			}

			//time.Sleep(1000 * time.Millisecond)
		}

	}
}
