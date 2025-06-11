package adapter

import (
	"encoding/json"

	"github.com/containerd/log"
)

func logRequest(endpoint string, request any) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		log.L.WithField("err", requestJSON).Debug(endpoint)
	}
	log.L.WithField("request", string(requestJSON)).Debug(endpoint)
}

func logResponse(endpoint string, response any) {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.L.WithField("err", responseJSON).Debug(endpoint)
	}
	log.L.WithField("response", string(responseJSON)).Debug(endpoint)
}
