package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/akamensky/argparse"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/nxadm/tail"
	"github.com/oschwald/maxminddb-golang"
	"github.com/vindolin/throttler"
)

// this struct stores the location of an IP address
type mmrecord struct {
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}

// parses an IP address from the beginning of a line
func parseIp(line string) (net.IP, error) {
	ipregexp := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	ipstr := ipregexp.FindString(line)

	if ipstr == "" {
		return nil, fmt.Errorf("failed to find IP address")
	}

	// parse the IP address
	ipAddr := net.ParseIP(ipstr)
	if ipAddr == nil {
		return nil, fmt.Errorf("failed to parse IP address")
	}

	return ipAddr, nil
}

func main() {
	// setup command line arguments
	parser := argparse.NewParser("run", "run the geolog websocket server")
	// mandatory arguments
	logFile := parser.String("l", "log_file",
		&argparse.Options{Required: true, Help: "log file to tail"})
	geoliteDb := parser.String("g", "geodb_file",
		&argparse.Options{Required: true, Help: "geolite db to use"})
	mqttServer := parser.String("m", "mqtt_server",
		&argparse.Options{Required: true, Help: "mqtt server to use"})

	// optional arguments
	mqttPort := parser.Int("p", "mqtt_port",
		&argparse.Options{Required: false, Help: "mqtt port to use", Default: 1884})
	mqttUsername := parser.String("u", "username",
		&argparse.Options{Required: false, Help: "mqtt username to use", Default: nil})
	mqttPassword := parser.String("P", "password",
		&argparse.Options{Required: false, Help: "mqtt password to use", Default: nil})

	topic := parser.String("t", "topic",
		&argparse.Options{Required: false, Help: "mqtt topic to use", Default: "location"})

	throttleDuration := parser.Int("T", "throttle_duration",
		&argparse.Options{Required: false, Help: "throttle in seconds", Default: 5})

	// parse the command line arguments
	err := parser.Parse(os.Args)
	if err != nil {
		log.Print(parser.Usage(err))
		os.Exit(1)
	}

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", *mqttServer, *mqttPort))
	opts.SetClientID("geolog")
	opts.SetUsername(*mqttUsername)
	opts.SetPassword(*mqttPassword)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// open the maxmind db
	gdb, err := maxminddb.Open(*geoliteDb)
	if err != nil {
		log.Fatalf("failed to open maxmind db: %v", err)
		os.Exit(1)
	}
	defer gdb.Close()

	// tail the log file
	tail, err := tail.TailFile(*logFile, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	})
	if err != nil {
		log.Fatal(err)
	}

	if throttleDuration == nil {
		log.Fatal("throttle duration is nil")
	}

	go func() {
		var throttler = throttler.New(time.Duration(*throttleDuration)*time.Second, 1000)

		// read lines from the log file
		for line := range tail.Lines {

			// parse the IP address from the line
			ip, _ := parseIp(line.Text)

			if ip == nil {
				continue
			}

			// ruhig Brauner!
			if !throttler.Allow(ip.String()) {
				continue
			}

			// lookup the IP address in the maxmind db
			var record mmrecord
			err = gdb.Lookup(ip, &record)
			if err != nil {
				log.Printf("failed to lookup ip %s: %v", ip, err)
				return
			}

			// format payload
			var payload = fmt.Sprintf(
				`["%s", %f, %f]`,
				ip, record.Location.Latitude,
				record.Location.Longitude)

			token := client.Publish(*topic, 0, false, payload)
			token.Wait()
			log.Println(payload)
		}
	}()

	// wait forever
	select {}
}
