/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"github.com/piprate/json-gold/ld"
	"github.com/trustbloc/edge-core/pkg/log"

	"github.com/trustbloc/orb/pkg/activitypub/service/vct"
)

var logger = log.New("proof-handler")

// New creates new proof handler.
func New(providers *Providers, vcChan chan *verifiable.Credential) *WitnessProofHandler {
	return &WitnessProofHandler{Providers: providers, vcCh: vcChan}
}

// Providers contains all of the providers required by the handler.
type Providers struct {
	Store         vcStore
	MonitoringSvc monitoringSvc
	DocLoader     ld.DocumentLoader
	WitnessStore  witnessStore
}

// WitnessProofHandler handles an anchor credential witness proof.
type WitnessProofHandler struct {
	*Providers
	vcCh chan *verifiable.Credential
}

type witnessStore interface {
	AddProof(vcID, witness string, p []byte) error
}

type vcStore interface {
	Get(id string) (*verifiable.Credential, error)
}

type monitoringSvc interface {
	Watch(anchorCredID string, endTime time.Time, proof []byte) error
}

// HandleProof handles proof.
func (h *WitnessProofHandler) HandleProof(witness *url.URL, anchorCredID string, startTime, endTime time.Time, proof []byte) error { //nolint:lll
	logger.Debugf("received request anchorCredID[%s] from witness[%s], proof: %s",
		anchorCredID, witness.String(), string(proof))

	err := h.MonitoringSvc.Watch(anchorCredID, endTime, proof)
	if err != nil {
		return fmt.Errorf("failed to setup monitoring for anchor credential[%s]: %w", anchorCredID, err)
	}

	var witnessProof vct.Proof

	err = json.Unmarshal(proof, &witnessProof)
	if err != nil {
		return fmt.Errorf("failed to unmarshal witness proof for anchor credential[%s]: %w", anchorCredID, err)
	}

	vc, err := h.Store.Get(anchorCredID)
	if err != nil {
		return fmt.Errorf("failed to retrieve anchor credential[%s]: %w", anchorCredID, err)
	}

	if len(vc.Proofs) > 1 {
		// TODO: issue-322 (handle multiple proofs - our witness policy is currently 1)
		logger.Debugf("Credential[%s] has already been witnessed, nothing to do", vc.ID)

		return nil
	}

	err = h.WitnessStore.AddProof(anchorCredID, witness.String(), proof)
	if err != nil {
		return fmt.Errorf("failed to add witness[%s] proof for credential[%s]: %w", witness.String(), anchorCredID, err)
	}

	vc.Proofs = append(vc.Proofs, witnessProof.Proof)

	h.vcCh <- vc

	return nil
}
