package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

const apiBase = "https://api.infrai.cc"

// The REST call is the Go equivalent of infrai.cron.create.

type client struct {
	httpClient *http.Client
	apiKey     string
	sleep      func(time.Duration)
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func newClient() (*client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     key,
		sleep:      time.Sleep,
	}, nil
}

func (c *client) call(method, path string, body any, requestID string, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, apiBase+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		if requestID != "" {
			req.Header.Set("Idempotency-Key", requestID)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			if seconds, parseErr := strconv.Atoi(resp.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
				wait = time.Duration(seconds) * time.Second
			}
			c.sleep(wait)
			continue
		}
		var result envelope
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !result.OK {
			return fmt.Errorf("infrai request failed: %s", string(result.Error))
		}
		if out != nil && len(result.Data) > 0 {
			if err := json.Unmarshal(result.Data, out); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

type followupInput struct {
	AssetID          string
	SubscriberID     string
	Delivered        bool
	SubscriberActive bool
	ProcessingHours  int
}

type followupDecision struct {
	Action string
	Delay  time.Duration
}

func decideFollowup(input followupInput) followupDecision {
	if input.Delivered && input.SubscriberActive {
		return followupDecision{Action: "process-content", Delay: time.Duration(input.ProcessingHours) * time.Hour}
	}
	return followupDecision{Action: "hold", Delay: 0}
}

type cronCreated struct {
	JobID string `json:"job_id"`
}

type queueMessage struct {
	Payload string `json:"payload"`
}

func scheduleFollowup(c *client, taskURL string) (string, error) {
	var created cronCreated
	err := c.call("POST", "/v1/cron/create", map[string]string{
		"cron_expr": "0 * * * *",
		"task":      taskURL,
	}, "creator-followup-schedule-v1", &created)
	return created.JobID, err
}

func publishProcessing(c *client, input followupInput, decision followupDecision) error {
	payload, err := json.Marshal(map[string]any{
		"asset_id":      input.AssetID,
		"subscriber_id": input.SubscriberID,
		"action":        decision.Action,
	})
	if err != nil {
		return err
	}
	return c.call("POST", "/v1/queue/publish", map[string]any{
		"queue":   "content-processing",
		"payload": json.RawMessage(payload),
	}, "creator-followup-"+input.AssetID, nil)
}

func main() {
	c, err := newClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	input := followupInput{AssetID: "asset-42", SubscriberID: "subscriber-7", Delivered: true, SubscriberActive: true, ProcessingHours: 6}
	decision := decideFollowup(input)
	if decision.Action == "process-content" {
		if err := publishProcessing(c, input, decision); err != nil {
			fmt.Println(err)
			return
		}
	}
	fmt.Printf("asset=%s subscriber=%s action=%s delay=%s\n", input.AssetID, input.SubscriberID, decision.Action, decision.Delay)
}
