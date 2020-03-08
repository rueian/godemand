package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rueian/godemand/types"
)

func NewHTTPClient(host string, info types.Client, client *http.Client) *HTTPClient {
	return &HTTPClient{
		host:   host,
		info:   info,
		client: client,
	}
}

type HTTPClient struct {
	host      string
	client    *http.Client
	info      types.Client
	requestAt time.Time
	servedAt  time.Time
}

var NotFoundError = errors.New("http status 404")

func (c *HTTPClient) RequestResource(ctx context.Context, poolID string) (resource types.Resource, err error) {
	c.requestAt = time.Now()
	for {
		res, err := c.postRetry(ctx, "/RequestResource", makeForm(poolID, "", c.info))
		if err != nil {
			return types.Resource{}, err
		}
		if err = json.Unmarshal(res, &resource); err != nil {
			return types.Resource{}, err
		}

		for {
			if resource.State == types.ResourceServing {
				c.servedAt = time.Now()
				return resource, err
			} else {
				time.Sleep(5 * time.Second)
			}
			res, err = c.postRetry(ctx, "/GetResource", makeForm(resource.PoolID, resource.ID, c.info))
			if errors.Is(err, NotFoundError) {
				break
			}
			if err != nil {
				return types.Resource{}, err
			}
			resource = types.Resource{}
			if err = json.Unmarshal(res, &resource); err != nil {
				return types.Resource{}, err
			}
		}
	}
}

func (c *HTTPClient) Heartbeat(ctx context.Context, resource types.Resource) (err error) {
	if c.info.Meta != nil {
		c.info.Meta["requestAt"] = c.requestAt
		c.info.Meta["servedAt"] = c.servedAt
	}

	_, err = c.postRetry(ctx, "/Heartbeat", makeForm(resource.PoolID, resource.ID, c.info))
	return err
}

func (c *HTTPClient) postRetry(ctx context.Context, endpoint string, form url.Values) (res []byte, err error) {
	var msg []string
	var resp *http.Response
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("errors: %s: %w", strings.Join(msg, "|"), ctx.Err())
		default:
		}

		resp, err = c.client.PostForm(c.host+endpoint, form)
		if err != nil {
			return nil, err
		}
		if resp.Body != nil {
			res, err = ioutil.ReadAll(resp.Body)
			resp.Body.Close()
		}
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return res, nil
			}
			if resp.StatusCode == 404 {
				return nil, fmt.Errorf("%s: %w", string(res), NotFoundError)
			}
		}
		if len(res) > 0 {
			msg = append(msg, string(res))
		}
		time.Sleep(1 * time.Second)
	}
}

func makeForm(poolID, id string, client types.Client) url.Values {
	cb, _ := json.Marshal(client)
	form := url.Values{}
	form.Add("poolID", poolID)
	form.Add("id", id)
	form.Add("client", string(cb))
	return form
}
