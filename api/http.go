package api

import (
	"encoding/json"
	"net/http"

	"github.com/rueian/godemand/plugin"
	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

func NewHTTPMux(s types.Service) *http.ServeMux {
	mux := &http.ServeMux{}
	mux.HandleFunc("/RequestResource", func(writer http.ResponseWriter, request *http.Request) {
		request.ParseForm()

		poolID := request.Form.Get("poolID")

		client, ok := handleClient(writer, request.Form.Get("client"))
		if !ok {
			return
		}

		res, err := s.RequestResource(poolID, client)
		if handleErr(writer, err) {
			return
		}

		writeRes(writer, res)
	})
	mux.HandleFunc("/GetResource", func(writer http.ResponseWriter, request *http.Request) {
		request.ParseForm()

		poolID := request.Form.Get("poolID")
		id := request.Form.Get("id")

		res, err := s.GetResource(poolID, id)
		if handleErr(writer, err) {
			return
		}

		writeRes(writer, res)
	})
	mux.HandleFunc("/Heartbeat", func(writer http.ResponseWriter, request *http.Request) {
		request.ParseForm()

		poolID := request.Form.Get("poolID")
		id := request.Form.Get("id")
		client, ok := handleClient(writer, request.Form.Get("client"))
		if !ok {
			return
		}

		err := s.Heartbeat(poolID, id, client)
		if handleErr(writer, err) {
			return
		}

		writer.WriteHeader(200)
	})
	return mux
}

func writeRes(w http.ResponseWriter, resource types.Resource) {
	ba, err := json.Marshal(resource)
	if err != nil {
		handleErr(w, err)
	} else {
		w.WriteHeader(200)
		w.Write(ba)
	}
}

func handleClient(w http.ResponseWriter, input string) (client types.Client, ok bool) {
	err := json.Unmarshal([]byte(input), &client)
	if err != nil {
		w.WriteHeader(422)
		w.Write([]byte("fail to parse the the client field"))
		return types.Client{}, false
	}
	return client, true
}

func handleErr(w http.ResponseWriter, err error) bool {
	if xerrors.Is(err, plugin.AcquireLaterErr) {
		w.WriteHeader(429)
	} else if xerrors.Is(err, ResourceNotFoundErr) {
		w.WriteHeader(404)
	} else if err != nil {
		w.WriteHeader(500)
	}
	if err != nil {
		w.Write([]byte(err.Error()))
		return true
	}
	return false
}
