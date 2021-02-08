/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package observer

import (
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/verifier"
	"github.com/hyperledger/aries-framework-go/pkg/doc/util"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/sidetree-core-go/pkg/api/operation"
	"github.com/trustbloc/sidetree-core-go/pkg/api/txn"
	"github.com/trustbloc/sidetree-core-go/pkg/mocks"
	"github.com/trustbloc/sidetree-core-go/pkg/versions/0_1/txnprocessor"

	orbtxn "github.com/trustbloc/orb/pkg/api/txn"
	"github.com/trustbloc/orb/pkg/txngraph"
)

func TestStartObserver(t *testing.T) {
	const (
		namespace1 = "ns1"
		namespace2 = "ns2"
	)

	t.Run("test channel close", func(t *testing.T) {
		sidetreeTxnCh := make(chan []string, 100)

		providers := &Providers{
			TxnProvider: mockLedger{registerForSidetreeTxnValue: sidetreeTxnCh},
		}

		o := New(providers)
		require.NotNil(t, o)

		o.Start()
		defer o.Stop()

		close(sidetreeTxnCh)
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("test success", func(t *testing.T) {
		sidetreeTxnCh := make(chan []string, 100)

		tp := &mocks.TxnProcessor{}

		pc := mocks.NewMockProtocolClient()
		pc.Protocol.GenesisTime = 1
		pc.Versions[0].TransactionProcessorReturns(tp)
		pc.Versions[0].ProtocolReturns(pc.Protocol)

		var txns []string
		txnGraph := txngraph.New(mocks.NewMockCasClient(nil), pubKeyFetcherFnc)

		payload1 := orbtxn.Payload{Namespace: namespace1, Version: 1, AnchorString: "1.address"}

		cid, err := txnGraph.Add(buildCredential(payload1))
		require.NoError(t, err)
		txns = append(txns, cid)

		payload2 := orbtxn.Payload{Namespace: namespace2, Version: 1, AnchorString: "2.address"}

		cid, err = txnGraph.Add(buildCredential(payload2))
		require.NoError(t, err)
		txns = append(txns, cid)

		providers := &Providers{
			TxnProvider:            mockLedger{registerForSidetreeTxnValue: sidetreeTxnCh},
			ProtocolClientProvider: mocks.NewMockProtocolClientProvider().WithProtocolClient(namespace1, pc),
			TxnGraph:               txnGraph,
		}

		o := New(providers)
		require.NotNil(t, o)

		o.Start()
		defer o.Stop()

		sidetreeTxnCh <- txns
		time.Sleep(200 * time.Millisecond)

		require.Equal(t, 1, tp.ProcessCallCount())
	})
}

func TestTxnProcessor_Process(t *testing.T) {
	t.Run("test error from txn operations provider", func(t *testing.T) {
		errExpected := fmt.Errorf("txn operations provider error")

		opp := &mockTxnOpsProvider{
			err: errExpected,
		}

		providers := &txnprocessor.Providers{
			OpStore:                   &mockOperationStore{},
			OperationProtocolProvider: opp,
		}

		p := txnprocessor.New(providers)
		err := p.Process(txn.SidetreeTxn{})
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
	})
}

type mockLedger struct {
	registerForSidetreeTxnValue chan []string
}

func (m mockLedger) RegisterForOrbTxn() <-chan []string {
	return m.registerForSidetreeTxnValue
}

type mockOperationStore struct {
	putFunc func(ops []*operation.AnchoredOperation) error
	getFunc func(suffix string) ([]*operation.AnchoredOperation, error)
}

func (m *mockOperationStore) Put(ops []*operation.AnchoredOperation) error {
	if m.putFunc != nil {
		return m.putFunc(ops)
	}

	return nil
}

func (m *mockOperationStore) Get(suffix string) ([]*operation.AnchoredOperation, error) {
	if m.getFunc != nil {
		return m.getFunc(suffix)
	}

	return nil, nil
}

type mockTxnOpsProvider struct {
	err error
}

func (m *mockTxnOpsProvider) GetTxnOperations(_ *txn.SidetreeTxn) ([]*operation.AnchoredOperation, error) {
	if m.err != nil {
		return nil, m.err
	}

	op := &operation.AnchoredOperation{
		UniqueSuffix: "abc",
	}

	return []*operation.AnchoredOperation{op}, nil
}

func buildCredential(payload orbtxn.Payload) *verifiable.Credential {
	const defVCContext = "https://www.w3.org/2018/credentials/v1"

	vc := &verifiable.Credential{
		Types:   []string{"VerifiableCredential"},
		Context: []string{defVCContext},
		Subject: payload,
		Issuer: verifiable.Issuer{
			ID: "http://peer1.com",
		},
		Issued: &util.TimeWithTrailingZeroMsec{Time: time.Now()},
	}

	return vc
}

var pubKeyFetcherFnc = func(issuerID, keyID string) (*verifier.PublicKey, error) {
	return nil, nil
}
