package twitchchatscraper

import (
	"github.com/sorcix/irc"
	"gopkg.in/olivere/elastic.v3"
	"time"

	log "github.com/cihub/seelog"
)

type ElasticBroker struct {
	inputChannel <-chan *irc.Message
	elastiClient *elastic.Client
}

type TwitchMessage struct {
	Channel   string
	Message   string
	From      string
	Timestamp time.Time `json:"@timestamp"`
}

func (e *ElasticBroker) Connect() chan<- *irc.Message {
	inputChannel := make(chan *irc.Message, 10000)
	e.inputChannel = inputChannel
	var clientError error
	e.elastiClient, clientError = elastic.NewClient(elastic.SetURL("http://192.168.1.110:9200"), elastic.SetSniff(false)) // Make configurable
	if clientError != nil {
		log.Errorf("Error connecting to elasticsearch: %s", clientError.Error())
	}

	go e.listenForMessages()
	return inputChannel
}

func (e *ElasticBroker) listenForMessages() {
	bulkRequest := e.elastiClient.Bulk()
	for {
		message := <-e.inputChannel
		twitchMessage := TwitchMessage{Channel: message.Params[0], Message: message.Trailing, From: message.User, Timestamp: time.Now()}
		bulkRequest.Add(elastic.NewBulkIndexRequest().Index("twitch").Type("chatmessage").Doc(twitchMessage))

		if bulkRequest.NumberOfActions() > 999 {
			log.Debugf("Applying %d bulk operations", bulkRequest.NumberOfActions())
			bulkRequest.Do()
		}
	}
}