package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type artifact struct {
	Timestep       string `json:"timestamp"`
	NodeID         string `json:"nodeId"`
	Initiator      string `json:"initiator"`
	RepositoryName string `json:"repositoryName"`
	Action         string `json:"action"`
	Component      struct {
		ID      string `json:"id"`
		Format  string `json:"format"`
		Name    string `json:"name"`
		Group   string `json:"group"`
		Version string `json:"version"`
	}
}

func nexusHandler(w http.ResponseWriter, r *http.Request, secretPath string, gitlab gitlab) {
	log.Printf("call to /nexus")

	w.Header().Set("Content-type", "application/json")

	// [improvement] use https://github.com/fsnotify/fsnotify/blob/master/example_test.go and keep secret in memory
	secret, err := ioutil.ReadFile(secretPath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("failed to load secret: %s", err)
		return
	}

	payload, err := extractPayload(secret, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("failed to parse input: %s", err)
		_, err = io.WriteString(w, "{\"error\":\"true\"}")
		if err != nil {
			log.Printf("failed to send respose: %s", err)
		}
		return
	}

	if payload.Action != "UPDATED" {
		w.WriteHeader(http.StatusOK)
		log.Printf("only allowing UPDATE action: action is %s", payload.Action)
		_, err = io.WriteString(w, "{\"error\":\"true\"}")
		if err != nil {
			log.Printf("failed to send respose: %s", err)
		}
		return
	}

	err = gitlab.notify(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed notify gitlab: %s", err)
		_, err = io.WriteString(w, "{\"error\":\"true\"}")
		if err != nil {
			log.Printf("failed to send respose: %s", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = io.WriteString(w, "{\"error\":\"false\"}")
	if err != nil {
		log.Printf("failed to send respose: %s", err)
		return
	}
}

func signBody(secret, body []byte) ([]byte, error) {
	computed := hmac.New(sha1.New, secret)
	_, err := computed.Write(body)
	if err != nil {
		return nil, fmt.Errorf("failed to sign body: %s", err)
	}
	return computed.Sum(nil), nil
}

func verifySignature(secret []byte, signature string, body []byte) (bool, error) {
	transmittedSignature := make([]byte, 20)
	_, err := hex.Decode(transmittedSignature, []byte(signature))
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %s", err)
	}
	computedSignature, err := signBody(secret, body)
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %s", err)
	}
	return hmac.Equal(transmittedSignature, computedSignature), nil
}

func extractPayload(secret []byte, req *http.Request) (artifact, error) {
	signature := req.Header.Get("x-nexus-webhook-signature")
	if len(signature) == 0 {
		return artifact{}, errors.New("no signature")
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return artifact{}, err
	}

	ok, err := verifySignature(secret, signature, body)
	if err != nil {
		return artifact{}, err
	}
	if !ok {
		return artifact{}, errors.New("invalid signature")
	}

	var a artifact
	err = json.Unmarshal(body, &a)
	if err != nil {
		return artifact{}, err
	}

	return a, nil
}
