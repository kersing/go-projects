package rn2483

import (
	"fmt"
	"github.com/tarm/serial"
	"log"
	"strconv"
	"strings"
	"time"
	"encoding/hex"
)

type Rn2483 struct {
	port *serial.Port
}

const (
	// Commands
	MAC_ABP_STR                 = "abp"
	MAC_ADR_STR                 = "adr"
	MAC_APP_EUI_KEY_STR         = "appeui"
	MAC_APP_KEY_STR             = "appkey"
	MAC_APP_SESSION_KEY_STR     = "appskey"
	MAC_CONFIRMED_STR           = "cnf "
	MAC_DEVADDR_STR             = "devaddr"
	MAC_DEV_EUI_KEY_STR         = "deveui"
	MAC_DR_STR                  = "dr"
	MAC_GET_STR                 = "mac get "
	MAC_JOIN_STR                = "mac join "
	MAC_NETWORK_SESSION_KEY_STR = "nwkskey"
	MAC_OTAA_STR                = "otaa"
	MAC_RX2_STR                 = "rx2"
	MAC_RX_STR                  = "mac_rx"
	MAC_SET_STR                 = "mac set "
	MAC_TX_STR                  = "mac tx "
	MAC_TX_OK_STR               = "mac_tx_ok"
	MAC_UNCONFIRMED_STR         = "uncnf "
	RADIO_POWER_STR             = "pwr"
	RADIO_SET_STR               = "radio set "
	RESTULT_ACCEPTED_STR        = "accepted"
	RESULT_INVALID_PARAM_STR    = "invalid_param"
	RESULT_OK_STR               = "ok"
	RETRIES_STR                 = "retx"
	SYS_GET_HWEUI_STR           = "sys get hweui"
	SYS_GET_STR                 = "sys get "
	SYS_RESET_STR               = "sys reset"
	SYS_STR                     = "sys "

	// Generic results
	RESULT_OK      = "ok"
	RESULT_INVALID = "invalid_param"

	// Results strings for transmit
	XMIT_RES_NOT_JOINED                  = "not_joined"
	XMIT_NO_FREE_CHANNEL                 = "no_free_ch"
	XMIT_SILENT                          = "silent"
	XMIT_FRAME_COUNTER_ERR_REJOIN_NEEDED = "frame_counter_err_rejoin_needed"
	XMIT_BUSY                            = "busy"
	XMIT_MAC_PAUSED                      = "mac_paused"
	XMIT_INVALID_DATA_LENGTH             = "invalid_data_len"
	XMIT_MAC_TX_OK                       = "mac_tx_ok"
	XMIT_MAC_ERR                         = "mac_err"
	XMIT_MAC_RX                          = "mac_rx"
)

const (
	// Result codes
	// Transmitted ok
	XMIT_OK = iota
	// Got data
	XMIT_DATA = iota
	// Transmit failed fataly for this packet
	XMIT_FAIL = iota
	// Transmit failed retry sending
	XMIT_RETRY = iota
	// Transmit failed, reset mac
	XMIT_RESET = iota
	// Transmit failed, rejoin required
	XMIT_REJOIN = iota
)

var xmit_response = map[string]int{
	RESULT_INVALID:                       XMIT_FAIL,
	XMIT_RES_NOT_JOINED:                  XMIT_REJOIN,
	XMIT_NO_FREE_CHANNEL:                 XMIT_RETRY,
	XMIT_SILENT:                          XMIT_RESET,
	XMIT_FRAME_COUNTER_ERR_REJOIN_NEEDED: XMIT_REJOIN,
	XMIT_BUSY:                XMIT_FAIL,
	XMIT_MAC_PAUSED:          XMIT_RESET,
	XMIT_INVALID_DATA_LENGTH: XMIT_FAIL,
	XMIT_MAC_ERR:             XMIT_RESET,
	XMIT_MAC_TX_OK:           XMIT_OK,
	XMIT_MAC_RX:              XMIT_OK,
}

var debug bool
var receivedData []byte

// Send outputs the supplied string to the serial port
func (rn *Rn2483) Send(cmd string) {
	rn.port.Write([]byte(cmd))
	if debug {
		fmt.Printf("%s", cmd)
	}
}

// debugPrint shows the character in a user friendly
// format.
func debugPrint(c byte) {
	if (c < 32 || c > 122) && c != 10 && c != 13 {
		fmt.Printf(" %02x ", c)
	} else {
		fmt.Printf("%c", c)
	}
}

// expect reads data from the serial port until the 'expected'
// string is matched or a timeout occurs. Optionally the
// remainder of the line (up till LF) is discarded.
// returns whether a timeout occurred.
func (rn *Rn2483) expect(str string, timeout int32, readline bool) bool {
	buf := make([]byte, 1)
	matched := 0
	l := len(str)

	if debug {
		fmt.Printf("Expect: >%s<\n", str)
	}
	for timeout > 0 && matched < l {
		n, _ := rn.port.Read(buf)
		if n < 1 {
			timeout -= 5
		} else {
			if debug {
				debugPrint(byte(buf[0] & 0xff))
			}
			if str[matched] == byte(buf[0]&0xff) {
				matched++
			} else {
				matched = 0
			}
		}
	}

	if matched >= l {
		if readline {
			buf[0] = 0
			for timeout > 0 && buf[0] != '\n' {
				n, _ := rn.port.Read(buf)
				if n < 1 {
					timeout -= 5
				} else {
					if debug {
						debugPrint(byte(buf[0] & 0xff))
					}
				}
			}
		}
		return true
	}
	return false
}

// InitRn2483 initializes the serial port to the rn2483 module
func InitRn2483(portName string, d bool) *Rn2483 {
	debug = d
	config := &serial.Config{Name: portName, Baud: 57600, ReadTimeout: time.Millisecond * 10}
	p, err := serial.OpenPort(config)

	if err != nil {
		log.Fatalf("Open RN2483 failed: %s\n", err)
	}
	return &Rn2483{port: p}
}

// InitAbp initializes the module and connects to LoRaWAN using the
// supplied device address and keys. Returns true if all commands
// were accepted by the module. (DOES NOT INDICATE A NETWORK WAS FOUND)
func (rn *Rn2483) JoinAbp(devAdr string, networkSessionkey string, appSessionkey string, adaptiveRate bool, dataRate int) bool {
	result := rn.Reset(true)
	if result {
		rn.Send(MAC_SET_STR + MAC_DEVADDR_STR + " " + devAdr + "\r\n")
		result = rn.expect("ok", 1000, true)
	}
	if result {
		rn.Send(MAC_SET_STR + MAC_APP_SESSION_KEY_STR + " " + appSessionkey + "\r\n")
		result = rn.expect("ok", 1000, true)
	}
	if result {
		rn.Send(MAC_SET_STR + MAC_NETWORK_SESSION_KEY_STR + " " + networkSessionkey + "\r\n")
		result = rn.expect("ok", 1000, true)
	}
	if result {
		rn.Send(MAC_SET_STR + MAC_ADR_STR + " ")
		if adaptiveRate {
			rn.Send("on")
		} else {
			rn.Send("off")
		}
		rn.Send("\r\n")
		result = rn.expect("ok", 1000, true)
	}
	if result && dataRate >= 0 {
		rn.Send(MAC_SET_STR + MAC_DR_STR + " " + strconv.Itoa(dataRate) + "\r\n")
		result = rn.expect("ok", 1000, true)
	}
	if result {
		rn.Send(MAC_JOIN_STR + MAC_ABP_STR + "\r\n")
		result = rn.expect("ok", 1000, true)
		if result {
			result = rn.expect("accepted", 15000, true)
		}
	}
	return result
}

// flushInput removes all remaining input from the serial port buffers
func (rn *Rn2483) flushInput() {
	buf := make([]byte, 1)
	for {
		n, _ := rn.port.Read(buf)
		if n < 1 {
			break
		}
	}
}

// Read line from serial port Returns string, timeout (bool)
func (rn *Rn2483) ReadLine(timeout int32) (string, bool) {
	buf := make([]byte, 1)
	input := make([]byte, 0, 255)
	eol := "\r\n"
	var matched int

	if debug {
		fmt.Println("ReadLine:")
	}

	// read until cr+lf found or timeout occurs
	for timeout > 0 && matched < len(eol) {
		n, _ := rn.port.Read(buf)
		if n < 1 {
			timeout -= 10
		} else {
			if debug {
				debugPrint(byte(buf[0] & 0xff))
			}
			if eol[matched] == byte(buf[0]&0xff) {
				input = append(input, byte(buf[0]))
				matched++
			} else {
				input = append(input, byte(buf[0]))
				matched = 0
			}
		}
	}
	if timeout <= 0 {
		if debug {
			fmt.Printf("\nTimeout waiting for end of line %d\n", timeout)
		}
	}

	str := string(input[:len(input)-2])
	if debug {
		fmt.Printf("Got line >%q<\n", str)
	}
	return str, (timeout <= 0)
}

// Read result from serial port and compare to possible values. Returns port, data, error
func (rn *Rn2483) ReadResult(timeout int32) (int, []byte, error) {
	buf := make([]byte, 1)
	input := make([]byte, 0, 255)
	eol := "\r\n"
	var matched int

	if debug {
		fmt.Println("ReadResult:")
	}

	// read until cr+lf found or timeout occurs
	for timeout > 0 && matched < len(eol) {
		n, _ := rn.port.Read(buf)
		if n < 1 {
			timeout -= 5
		} else {
			if debug {
				debugPrint(byte(buf[0] & 0xff))
			}
			if eol[matched] == byte(buf[0]&0xff) {
				input = append(input, byte(buf[0]))
				matched++
			} else {
				input = append(input, byte(buf[0]))
				matched = 0
			}
		}
	}
	str := string(input[:len(input)-2])

	if timeout <= 0 {
		if debug {
			fmt.Printf("\nTimeout waiting for end of line %d\n", timeout)
		}
		return str, NewError("Timeout waiting for end of line", XMIT_FAIL)
	}

	if debug {
		fmt.Printf("Got line >%q<\n", str)
	}

	if strings.Contains(str, XMIT_MAC_RX) {
		// Got return data
		sep := strings.Index(str, " ")
		if sep < 0 {
			return 0, []byte{}, error.New("Port not found")
		}

		// Extract port number
		port := strconv.Atoi(str[:sep])
		bytes := make([]byte, 0, 10)
		len, error := hex.Decode(bytes, []byte(str[sep:]))
		if len <= 0 || error != nil {
			return 0, []byte{}, error
		}

		return port, bytes, nil
	}

	// Transmission successfull?
	if strings.Contains(str, XMIT_MAC_TX_OK) {
		return 0, []byte{}, nil
	}

	return 0, []byte{}, NewError(str, xmit_response[str])
}

// Reset sends the 'warm' reset string to the module and optionally checks
// for a response. Returns true if valid response received or response is
// ignored.
func (rn *Rn2483) Reset(checkResult bool) bool {
	rn.Send(SYS_RESET_STR + "\r\n")
	if checkResult {
		return rn.expect("RN2483", 10000, true)
	} else {
		time.Sleep(1 * time.Second)
		rn.flushInput()
	}
	return true
}

// Transmit sends the supplied data to the network. Returns
// the port number and data if data received, (0 and empty
// slice if not) and possible error (nil if no error)
func (rn *Rn2483) Transmit(port int, confirmed bool, data []byte) (int, []byte, error) {
	rn.Send("mac tx ")
	if confirmed {
		rn.Send("cnf")
	} else {
		rn.Send("uncnf")
	}
	rn.Send(fmt.Sprintf(" %d ", port))
	for i := 0; i < len(data); i++ {
		rn.Send(fmt.Sprintf("%02X", data[i]))
	}
	rn.Send("\r\n")
	if rn.expect("ok", 1000, true) {
		// confirmed tx command, now wait for result
		return rn.ReadResult(600000)
	}
	return 0, []byte{}, NewError("Transmit failed", XMIT_FAIL)
}
