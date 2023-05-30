package fuzzing

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/netrixframework/netrix/types"
)

type TLCGuider struct {
	stateMap map[int64]bool
	tlcAddr  string
}

var _ Guider = &TLCGuider{}

func NewTLCGuider(tlcAddr string) *TLCGuider {
	return &TLCGuider{
		tlcAddr:  tlcAddr,
		stateMap: make(map[int64]bool),
	}
}

func (t *TLCGuider) HaveNewState(trace *types.List[*SchedulingChoice], eventTrace *types.List[*types.Event]) bool {
	bs, err := json.Marshal(mapEventTrace(eventTrace))
	if err != nil {
		return false
	}
	resp, err := http.Post("http://"+t.tlcAddr+"/execute", "application/json", bytes.NewBuffer(bs))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	respS, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	tlcResp := &tlcResponse{}
	err = json.Unmarshal(respS, tlcResp)
	if err != nil {
		return false
	}

	haveNew := false
	for _, k := range tlcResp.Keys {
		if _, ok := t.stateMap[k]; !ok {
			haveNew = true
			t.stateMap[k] = true
		}
	}
	return haveNew
}

func (t *TLCGuider) Reset() {
	t.stateMap = make(map[int64]bool)
}

type tlcEvent struct {
	Name   string
	Params map[string]string
	Reset  bool
}

func mapEventTrace(events *types.List[*types.Event]) []tlcEvent {
	result := make([]tlcEvent, 0)
	for _, e := range events.Iter() {
		next := tlcEvent{
			Name:   e.TypeS,
			Params: make(map[string]string),
		}
		for k, v := range e.Params {
			next.Params[k] = v
		}
		result = append(result, next)
	}
	result = append(result, tlcEvent{Reset: true})
	return result
}

type tlcResponse struct {
	States []string
	Keys   []int64
}
