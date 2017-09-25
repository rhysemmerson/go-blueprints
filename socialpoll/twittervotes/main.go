package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"

	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/mgo.v2"
)

var (
	nsqdAddress *string
	mgoAddress  *string
)

func main() {
	// Handle stopping the services
	var stoplock sync.Mutex // protects stop
	stop := false
	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	go func() {
		<-signalChan
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		log.Println("Stopping...")
		stopChan <- struct{}{}
		closeConn()
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// parse flags
	nsqdAddress = flag.String("nsqd-http-address", "localhost:4161", "The address of the nsq daemon")
	mgoAddress = flag.String("db-address", "localhost", "Address of the database")

	flag.Parse()

	// Connect to db
	if err := dialdb(); err != nil {
		log.Fatalln("Failed to connect to MongoDB:", err)
	}
	defer closedb()

	// start the services
	votes := make(chan string)
	publisherStoppedChan := publishVotes(votes)
	twitterStoppedChan := startTwitterStream(stopChan, votes)
	go func() {
		for {
			// wait 1 minute then close the connection
			time.Sleep(1 * time.Minute)
			closeConn()
			stoplock.Lock()
			if stop {
				// if stop,
				stoplock.Unlock()
				return
			}
			stoplock.Unlock()
		}
	}()
	<-twitterStoppedChan   // wait until twitter stream stops
	close(votes)           // close votes chan to stop publishing to NSQ
	<-publisherStoppedChan // wait for publisher to stop
}

// publishVotes listens for votes received from the twitter api
// and publishes them to our NSQ instance
func publishVotes(votes <-chan string) <-chan struct{} {
	stopchan := make(chan struct{}, 1)
	pub, err := nsq.NewProducer(*nsqdAddress, nsq.NewConfig())
	if err != nil {
		log.Fatal("Could not connect to NSQ:", err)
	}
	go func() {
		for vote := range votes {
			pub.Publish("votes", []byte(vote))
		}
		log.Println("Publisher: Stopping")
		pub.Stop()
		log.Println("Publisher: Stopped")
		stopchan <- struct{}{}
	}()
	return stopchan
}

type poll struct {
	Options []string
}

func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
}

var db *mgo.Session

func dialdb() error {
	var err error
	log.Println("dialing mongodb:", mgoAddress)
	db, err = mgo.Dial(*mgoAddress)
	return err
}

func closedb() {
	db.Close()
	log.Println("closed database connection")
}
