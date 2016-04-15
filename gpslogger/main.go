// Copyright © 2016 by Jac Kersing
// Use of this source code is governed by the MIT license that can be found in the LICENSE file at the top
// of the github repository for this source

package main

import (
	"fmt"
	"strconv"
	"log"
	"time"
	"encoding/csv"
	"os"
	"flag"
	"github.com/kersing/go-projects/rn2483"
)

var counter int16
var gpsport string
var rn2483port string
var nodeId string
var sleepTime int
var loramodule *rn2483.Rn2483

func init() {
	flag.StringVar(&gpsport, "gps", "/dev/ttyUSB2", "GPS port")
	flag.StringVar(&rn2483port, "rn2483", "/dev/ttyUSB1", "RN2483 port")
	flag.IntVar(&sleepTime,"sleep", 45, "Time between transmissions")
	flag.StringVar(&nodeId,"node","020129FF", "Node ID")
}

func initLora() {
	loramodule = rn2483.InitRn2483(rn2483port,true)
	fmt.Println("Init RN2483")
	for {
		success := loramodule.JoinAbp(nodeId, "2B7E151628AED2A6ABF7158809CF4F3C", "2B7E151628AED2A6ABF7158809CF4F3C", false, 5)
		if success {
			break
		}
		fmt.Println("Init failed, wait 5 seconds and retry")
		time.Sleep(5 * time.Second)
	}
}

func main() {
	flag.Parse()
	initLora()

	go gps(gpsport)
	csvdata := make([]string, 6, 6)
	var xmitdata string
	t := time.Now()
	filename := fmt.Sprintf("%d%02d%02d%02d%02d.csv", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error opening output %s: %s",filename,err)
	}
	defer f.Close()
	w := csv.NewWriter(f)

	for {
		// Loop forever
		fmt.Println("Sleep",sleepTime, "seconds")
		time.Sleep(time.Duration(sleepTime) * time.Second)
		fmt.Printf("Time: %s, position: %f, %f +- %f (%s,%d)\n", coords.time, coords.lat, coords.long, coords.hdop, coords.fix, coords.sats)
		csvdata[0] = strconv.FormatInt(int64(counter),10)
		csvdata[1] = coords.time
		csvdata[2] = strconv.FormatFloat(float64(coords.lat), 'f', 6, 64)
		csvdata[3] = strconv.FormatFloat(float64(coords.long), 'f', 6, 64)
		csvdata[4] = coords.fix
		csvdata[5] = strconv.Itoa(coords.sats)
		w.Write(csvdata)
		w.Flush()
		counter++

		//xmitdata[0] = byte((counter & 0xff) >> 8)
		//xmitdata[1] = byte(counter & 0xff)
		xmitdata = fmt.Sprintf("%s,%s",csvdata[2],csvdata[3])
		_, _, error := loramodule.Transmit(1,false,[]byte(xmitdata))
		if error == nil {
			fmt.Println("Transmission successfull")
		} else {
			fmt.Println("Transmission failed:", error.Error())
			//log.Fatal("Exit")
			if rerr, ok := error.(rn2483.Error); ok && rerr.RetryAble() {
				fmt.Println("Reset RN2483")
				// try to recover by resetting the RN2483
				initLora()
			}
		}

		//loramodule.Send(rn2483.MAC_GET_STR + "status\r\n")
		//res, _ := loramodule.ReadLine(10000)
		//bin, _ := strconv.Atoi(res)
		//fmt.Printf("Result: %b\n",bin)
	}
}
