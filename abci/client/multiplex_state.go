package abcicli

import (
	"crypto/sha256"
	"encoding/json"
	dbm "github.com/cometbft/cometbft-db"
	"sort"
)

const stateKey = "multiplex_state"

type multiplexState struct {
	db        dbm.DB
	Size      int64             `json:"size"`
	Height    int64             `json:"height"`
	AppHashes map[string][]byte `json:"app_hash"`
}

func (s multiplexState) GenerateAppHash() []byte {
	appNames := keys(s.AppHashes)
	sort.Strings(appNames)

	var combined []byte
	for _, name := range appNames {
		combined = append(combined, s.AppHashes[name]...)
	}

	digest := sha256.New()
	digest.Write(combined)
	return digest.Sum(nil)
}

func loadMultiplexState(db dbm.DB) (multiplexState, error) {
	state := multiplexState{
		db: db,
	}

	stateBytes, err := db.Get([]byte(stateKey))
	if err != nil {
		return multiplexState{}, err
	}

	if len(stateBytes) == 0 {
		state.AppHashes = make(map[string][]byte)
		return state, nil
	}

	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return multiplexState{}, err
	}

	return state, nil
}

func (s *multiplexState) save() error {
	stateBytes, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return s.db.Set([]byte(stateKey), stateBytes)
}

func keys[T comparable, U any](m map[T]U) []T {
	keys := make([]T, len(m))

	i := 0
	for key := range m {
		keys[i] = key
		i++
	}

	return keys
}
