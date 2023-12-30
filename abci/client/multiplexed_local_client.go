package abcicli

import (
	"context"
	"errors"
	"fmt"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/version"
)

type multiplexedLocalClient struct {
	service.BaseService
	state multiplexState
	apps  map[string]types.Application

	mu sync.Mutex
	Callback
}

var _ Client = (*multiplexedLocalClient)(nil)

func NewMultiplexedLocalClient(db dbm.DB, apps map[string]types.Application) (Client, error) {
	state, err := loadMultiplexState(db)
	if err != nil {
		return nil, err
	}

	client := &multiplexedLocalClient{
		apps:  apps,
		state: state,
		mu:    sync.Mutex{},
	}

	client.BaseService = *service.NewBaseService(nil, "multiplexedLocalClient", client)
	return client, nil
}

func (m *multiplexedLocalClient) Info(ctx context.Context, req *types.RequestInfo) (*types.ResponseInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if ok {
		// Defer call
		res, err := app.Info(ctx, req)
		if err != nil {
			return nil, err
		}

		// Update AppHash from deferred call
		m.state.AppHashes[req.App] = res.LastBlockAppHash

		return &types.ResponseInfo{
			Data:             res.Data,
			Version:          version.ABCIVersion,
			AppVersion:       res.AppVersion,
			LastBlockHeight:  m.state.Height,
			LastBlockAppHash: m.state.GenerateAppHash(),
		}, nil
	} else {
		data, err := json.Marshal(keys(m.apps))
		if err != nil {
			return nil, err
		}

		return &types.ResponseInfo{
			Data:             string(data),
			Version:          version.ABCIVersion,
			AppVersion:       1,
			LastBlockHeight:  m.state.Height,
			LastBlockAppHash: m.state.GenerateAppHash(),
		}, nil
	}
}

func (m *multiplexedLocalClient) Query(ctx context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.Query(ctx, req)
}

func (m *multiplexedLocalClient) CheckTx(ctx context.Context, req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.CheckTx(ctx, req)
}

func (m *multiplexedLocalClient) InitChain(ctx context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, app := range m.apps {
		res, err := app.InitChain(ctx, req)
		if err != nil {
			return nil, err
		}

		m.state.AppHashes[name] = res.AppHash
	}

	return &types.ResponseInitChain{}, nil

	//app, ok := m.apps[req.App]
	//if !ok {
	//	return nil, fmt.Errorf("unknown app: %s", req.App)
	//}
	//
	//// Defer call
	//return app.InitChain(ctx, req)
}

func (m *multiplexedLocalClient) PrepareProposal(ctx context.Context, req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Do not modify proposed transaction group, simply return proposed set
	return &types.ResponsePrepareProposal{Txs: req.Txs}, nil

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.PrepareProposal(ctx, req)
}

func (m *multiplexedLocalClient) ProcessProposal(ctx context.Context, req *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Accept the proposed transaction group, no reason to reject
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.ProcessProposal(ctx, req)
}

func (m *multiplexedLocalClient) FinalizeBlock(ctx context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	fmt.Println(1)
	m.mu.Lock()
	fmt.Println(2)
	defer m.mu.Unlock()

	fmt.Println(3)
	if len(req.Txs) == 0 {
		return &types.ResponseFinalizeBlock{
			Events:                nil,
			TxResults:             nil,
			ValidatorUpdates:      nil,
			ConsensusParamUpdates: nil,
			AppHash:               m.state.GenerateAppHash(),
		}, nil
	}

	fmt.Println(4)
	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}
	fmt.Println(5)

	// Defer call
	res, err := app.FinalizeBlock(ctx, req)
	fmt.Println(6)
	if err != nil {
		return nil, err
	}
	fmt.Println(7)

	m.state.Height = req.Height
	m.state.AppHashes[req.App] = res.AppHash
	fmt.Println(8)
	return res, nil
}

func (m *multiplexedLocalClient) ExtendVote(ctx context.Context, req *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	return &types.ResponseExtendVote{}, nil

	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.ExtendVote(ctx, req)
}

func (m *multiplexedLocalClient) VerifyVoteExtension(ctx context.Context, req *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	return &types.ResponseVerifyVoteExtension{}, nil

	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.VerifyVoteExtension(ctx, req)
}

func (m *multiplexedLocalClient) Commit(ctx context.Context, req *types.RequestCommit) (*types.ResponseCommit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, app := range m.apps {
		_, err := app.Commit(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	resp := &types.ResponseCommit{}
	if 0 > 0 && m.state.Height >= 0 {
		resp.RetainHeight = m.state.Height - 0 + 1
	}

	return resp, nil
	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	if err := m.state.save(); err != nil {
		return nil, err
	}

	// Defer call
	return app.Commit(ctx, req)
}

func (m *multiplexedLocalClient) ListSnapshots(ctx context.Context, req *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.ListSnapshots(ctx, req)
}

func (m *multiplexedLocalClient) OfferSnapshot(ctx context.Context, req *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.OfferSnapshot(ctx, req)
}

func (m *multiplexedLocalClient) LoadSnapshotChunk(ctx context.Context, req *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.LoadSnapshotChunk(ctx, req)
}

func (m *multiplexedLocalClient) ApplySnapshotChunk(ctx context.Context, req *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	return app.ApplySnapshotChunk(ctx, req)
}

func (m *multiplexedLocalClient) Error() error {
	return nil
}

func (m *multiplexedLocalClient) Flush(_ context.Context) error {
	return nil
}

func (m *multiplexedLocalClient) Echo(_ context.Context, msg string) (*types.ResponseEcho, error) {
	return &types.ResponseEcho{Message: msg}, nil
}

func (m *multiplexedLocalClient) SetResponseCallback(callback Callback) {
	m.mu.Lock()
	m.Callback = callback
	m.mu.Unlock()
}

func (m *multiplexedLocalClient) CheckTxAsync(ctx context.Context, req *types.RequestCheckTx) (*ReqRes, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.apps[req.App]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", req.App)
	}

	// Defer call
	res, err := app.CheckTx(ctx, req)
	if err != nil {
		return nil, err
	}

	if m.Callback != nil {
		genericReq := types.ToRequestCheckTx(req)
		genericRes := types.ToResponseCheckTx(res)

		m.Callback(genericReq, genericRes)

		reqRes := NewReqRes(genericReq)
		reqRes.Response = genericRes
		reqRes.callbackInvoked = true
		return reqRes, nil
	} else {
		return nil, errors.New("m.Callback was nil")
	}
}
