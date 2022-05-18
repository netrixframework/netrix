package types

import (
	"encoding/json"
	"fmt"
	"time"
)

type ReplicaTimeout struct {
	Replica  ReplicaID     `json:"replica"`
	Type     string        `json:"type"`
	Duration time.Duration `json:"duration"`
}

func (t *ReplicaTimeout) Key() string {
	return fmt.Sprintf("%s_%s_%s", t.Replica, t.Type, t.Duration.String())
}

func (t *ReplicaTimeout) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"replica":  string(t.Replica),
		"type":     t.Type,
		"duration": t.Duration.String(),
	})
}

func (t *ReplicaTimeout) Eq(other *ReplicaTimeout) bool {
	return t.Replica == other.Replica && t.Type == other.Type && t.Duration == other.Duration
}

func TimeoutFromParams(replica ReplicaID, params map[string]string) (*ReplicaTimeout, bool) {
	t := &ReplicaTimeout{
		Replica: replica,
	}
	ttype, ok := params["type"]
	if !ok {
		return nil, false
	}
	t.Type = ttype
	durS, ok := params["duration"]
	dur, err := time.ParseDuration(durS)
	if !ok || err != nil {
		return nil, false
	}
	t.Duration = dur
	return t, true
}
