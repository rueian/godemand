package client

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

func NewHTTPClient(host string, info types.Client, client *http.Client) *HTTPClient {
	return &HTTPClient{
		host:   host,
		info:   info,
		client: client,
	}
}

type HTTPClient struct {
	host   string
	client *http.Client
	info   types.Client
}

func (c *HTTPClient) RequestResource(ctx context.Context, poolID string) (resource types.Resource, err error) {
	res, err := c.postRetry(ctx, "/RequestResource", makeForm(poolID, "", c.info))
	if err != nil {
		return types.Resource{}, err
	}
	if err = json.Unmarshal(res, &resource); err != nil {
		return types.Resource{}, err
	}

	for {
		if resource.State != types.ResourceRunning {
			time.Sleep(5 * time.Second)
		} else {
			break
		}
		res, err = c.postRetry(ctx, "/GetResource", makeForm(resource.PoolID, resource.ID, c.info))
		if err != nil {
			return types.Resource{}, err
		}
		resource = types.Resource{}
		if err = json.Unmarshal(res, &resource); err != nil {
			return types.Resource{}, err
		}
	}

	_, err = c.postRetry(ctx, "/Heartbeat", makeForm(resource.PoolID, resource.ID, c.info))
	return resource, err
}

func (c *HTTPClient) Heartbeat(ctx context.Context, resource types.Resource) (err error) {
	_, err = c.postRetry(ctx, "/Heartbeat", makeForm(resource.PoolID, resource.ID, c.info))
	return err
}

func (c *HTTPClient) postRetry(ctx context.Context, endpoint string, form url.Values) (res []byte, err error) {
	var msg []string
	var resp *http.Response
	for {
		select {
		case <-ctx.Done():
			return nil, xerrors.Errorf("errors: %s: %w", strings.Join(msg, "|"), ctx.Err())
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
				return nil, errors.New(string(res))
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