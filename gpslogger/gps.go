package main

import "fmt"
import "strconv"
import "github.com/buxtronix/golang/nmea/src/nmea"
import "github.com/tarm/serial"

var coords struct {
	time string
	lat  nmea.LatLong
	long nmea.LatLong
	sats int
	hdop float64
	fix  string
}

func readline(port *serial.Port) string {
	line := make([]byte, 0, 255)
	buf := make([]byte, 1)
//	var index int

	// Wait for start char ($)

	for {
		n, _ := port.Read(buf)
		if n > 0 {
			if buf[0] == '$' {
				line = append(line, buf[0])
//				line[index] = buf[0]
//				index++
				break
			}
		}
	}

	// Get rest of line
	for {
		n, _ := port.Read(buf)
		if n > 0 {
			if buf[0] == '\n' || buf[0] == '\r' {
				return string(line)
			} else {
				line = append(line, buf[0])
//				line[index] = buf[0]
//				index++
			}
		}
	}
	return "AA"
}

func gps(portName string) {
	// Open serial port
	config := &serial.Config{Name: portName, Baud: 9600}
	port, err := serial.OpenPort(config)
	if err != nil {
		fmt.Printf("Open failed: %s\n", err)
		return
	}

	for {
		sent := readline(port)
		s, err := nmea.Parse(sent)
		if err != nil {
			fmt.Printf("Error parsing: %s\n", err)
			return
		}

		switch s.(type) {
		case nmea.GPGGA:
			p, _ := s.(nmea.GPGGA)
			coords.time = p.Time
			coords.lat = p.Latitude
			coords.long = p.Longitude
			coords.sats, _ = strconv.Atoi(p.NumSatellites)
			coords.hdop, _ = strconv.ParseFloat(p.HDOP, 32)
			coords.fix = p.FixQuality
			//fmt.Printf("Quality: %s\n",p.FixQuality)
			//fmt.Printf("Position: %f, %f\n",p.Latitude, p.Longitude)
		}
	}
}
