package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/rueian/godemand/plugin"
	"github.com/rueian/godemand/types"
)

//go:generate go build -o puppet .

func main() {
	// print all envs for testing
	fmt.Println(strings.Join(os.Environ(), " "))

	if err := plugin.Serve(context.Background(), &PuppetController{}); err != nil {
		panic(err)
	}
}

type PuppetController struct{}

func (*PuppetController) FindResource(pool types.ResourcePool, params map[string]interface{}) (res types.Resource, err error) {
	if errMsg, ok := params["err"]; ok {
		return res, errors.New(errMsg.(string))
	}

	var resID string
	if retID, ok := params["ret"]; ok {
		return pool.Resources[retID.(string)], nil
	} else {
		for _, res := range pool.Resources {
			return res, nil
		}

		resID = strconv.Itoa(rand.Int())
	}

	return types.Resource{ID: resID}, nil
}

func (*PuppetController) SyncResource(resource types.Resource, params map[string]interface{}) (res types.Resource, err error) {
	if errMsg, ok := params["err"]; ok {
		return res, errors.New(errMsg.(string))
	}
	if state, ok := params["state"]; ok {
		resource.State = types.ResourceState(state.(float64))
	}
	return resource, err
}
