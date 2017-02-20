package opentimestamps

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/Sirupsen/logrus"
)

const userAgent = "go-opentimestamps"

const dumpResponse = false

type RemoteCalendar struct {
	baseURL string
	client  *http.Client
	log     *logrus.Logger
}

func NewRemoteCalendar(baseURL string) (*RemoteCalendar, error) {
	// FIXME remove this
	if baseURL == "localhost" {
		baseURL = "http://localhost:14788"
	}
	// TODO validate url
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &RemoteCalendar{
		baseURL,
		http.DefaultClient,
		logrus.New(),
	}, nil
}

// Check response status, return informational error message if
// status is not `200 OK`.
func checkStatusOK(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	errMsg := fmt.Sprintf("unexpected response: %q", resp.Status)
	if resp.Body == nil {
		return fmt.Errorf("%s (body=nil)", errMsg)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s (bodyErr=%v)", errMsg, err)
	} else {
		return fmt.Errorf("%s (body=%q)", errMsg, bodyBytes)
	}
}

func (c *RemoteCalendar) do(r *http.Request) (*http.Response, error) {
	r.Header.Add("Accept", "application/vnd.opentimestamps.v1")
	r.Header.Add("User-Agent", userAgent)
	c.log.Debugf("> %s %s", r.Method, r.URL)
	resp, err := c.client.Do(r)
	if err != nil {
		c.log.Errorf("> %s %s error: %v", r.Method, r.URL, err)
		return resp, err
	}
	c.log.Debugf("< %s %s - %v", r.Method, r.URL, resp.Status)
	if dumpResponse {
		bytes, err := httputil.DumpResponse(resp, true)
		if err == nil {
			c.log.Debugf("response dump:%s ", bytes)
		}
	}
	return resp, err
}

func (c *RemoteCalendar) url(path string) string {
	return c.baseURL + path
}

func (c *RemoteCalendar) Submit(digest []byte) (*Timestamp, error) {
	body := bytes.NewBuffer(digest)
	req, err := http.NewRequest("POST", c.url("digest"), body)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected 200, got %v", resp.Status)
	}
	return NewTimestampFromReader(resp.Body, digest)
}

func (c *RemoteCalendar) GetTimestamp(commitment []byte) (*Timestamp, error) {
	url := c.url("timestamp/" + hex.EncodeToString(commitment))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if err := checkStatusOK(resp); err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	return NewTimestampFromReader(resp.Body, commitment)
}

type PendingTimestamp struct {
	Timestamp          *Timestamp
	PendingAttestation *pendingAttestation
}

func (p PendingTimestamp) Upgrade() (*Timestamp, error) {
	cal, err := NewRemoteCalendar(p.PendingAttestation.uri)
	if err != nil {
		return nil, err
	}
	return cal.GetTimestamp(p.Timestamp.Message)
}

func PendingTimestamps(ts *Timestamp) (res []PendingTimestamp) {
	ts.Walk(func(ts *Timestamp) {
		for _, att := range ts.Attestations {
			p, ok := att.(*pendingAttestation)
			if !ok {
				continue
			}
			attCopy := *p
			res = append(res, PendingTimestamp{ts, &attCopy})
		}
	})
	return
}
