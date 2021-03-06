/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resthandler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/trustbloc/orb/pkg/activitypub/store/spi"
	"github.com/trustbloc/orb/pkg/activitypub/store/storeutil"
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

// NewActivity returns a new 'activities/{id}' REST handler that retrieves a single activity by ID.
func NewActivity(cfg *Config, activityStore spi.Store, verifier signatureVerifier) *Activity {
	h := &Activity{}
	h.handler = newHandler(ActivitiesPath, cfg, activityStore, h.handle, verifier)

	return h
}

// NewOutbox returns a new 'outbox' REST handler that retrieves a service's outbox.
func NewOutbox(cfg *Config, activityStore spi.Store, verifier signatureVerifier) *Activities {
	return NewActivities(OutboxPath, spi.Outbox, cfg, activityStore,
		getObjectIRI(cfg.ObjectIRI), getID("outbox"), verifier)
}

// NewInbox returns a new 'inbox' REST handler that retrieves a service's inbox.
func NewInbox(cfg *Config, activityStore spi.Store, verifier signatureVerifier) *Activities {
	return NewActivities(InboxPath, spi.Inbox, cfg, activityStore,
		getObjectIRI(cfg.ObjectIRI), getID("inbox"), verifier)
}

// NewShares returns a new 'shares' REST handler that retrieves an object's 'Announce' activities.
func NewShares(cfg *Config, activityStore spi.Store, verifier signatureVerifier) *Activities {
	return NewActivities(SharesPath, spi.Share, cfg, activityStore,
		getObjectIRIFromParam(cfg.ObjectIRI), getID("shares"), verifier)
}

// NewLikes returns a new 'likes' REST handler that retrieves an object's 'Like' activities.
func NewLikes(cfg *Config, activityStore spi.Store, verifier signatureVerifier) *Activities {
	return NewActivities(LikesPath, spi.Like, cfg, activityStore,
		getObjectIRIFromParam(cfg.ObjectIRI), getID("likes"), verifier)
}

// NewLiked returns a new 'liked' REST handler that retrieves a service's 'Like' activities, i.e. the Like
// activities that were posted by the given service.
func NewLiked(cfg *Config, activityStore spi.Store, verifier signatureVerifier) *Activities {
	return NewActivities(LikedPath, spi.Liked, cfg, activityStore,
		getObjectIRI(cfg.ObjectIRI), getID("liked"), verifier)
}

type getIDFunc func(objectIRI *url.URL) (*url.URL, error)

type getObjectIRIFunc func(req *http.Request) (*url.URL, error)

// Activities implements a REST handler that retrieves activities.
type Activities struct {
	*handler

	refType      spi.ReferenceType
	getID        getIDFunc
	getObjectIRI getObjectIRIFunc
}

// NewActivities returns a new activities REST handler.
func NewActivities(path string, refType spi.ReferenceType, cfg *Config, activityStore spi.Store,
	getObjectIRI getObjectIRIFunc, getID getIDFunc, verifier signatureVerifier) *Activities {
	h := &Activities{
		refType:      refType,
		getID:        getID,
		getObjectIRI: getObjectIRI,
	}

	h.handler = newHandler(path, cfg, activityStore, h.handle, verifier)

	return h
}

func (h *Activities) handle(w http.ResponseWriter, req *http.Request) {
	ok, _, err := h.verifier.VerifyRequest(req)
	if err != nil {
		logger.Errorf("[%s] Error verifying HTTP signature: %s", h.endpoint, err)

		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !ok {
		logger.Infof("[%s] Invalid HTTP signature", h.endpoint)

		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	objectIRI, err := h.getObjectIRI(req)
	if err != nil {
		logger.Errorf("[%s] Error getting ObjectIRI: %s", h.endpoint, err)

		h.writeResponse(w, http.StatusInternalServerError, nil)

		return
	}

	id, err := h.getID(objectIRI)
	if err != nil {
		logger.Errorf("[%s] Error generating ID: %s", h.endpoint, err)

		h.writeResponse(w, http.StatusInternalServerError, nil)

		return
	}

	if h.isPaging(req) {
		h.handleActivitiesPage(w, req, objectIRI, id)
	} else {
		h.handleActivities(w, req, objectIRI, id)
	}
}

func (h *Activities) handleActivities(rw http.ResponseWriter, _ *http.Request, objectIRI, id *url.URL) {
	activities, err := h.getActivities(objectIRI, id)
	if err != nil {
		logger.Errorf("[%s] Error retrieving %s for object IRI [%s]: %s",
			h.endpoint, h.refType, objectIRI, err)

		h.writeResponse(rw, http.StatusInternalServerError, nil)

		return
	}

	activitiesCollBytes, err := h.marshal(activities)
	if err != nil {
		logger.Errorf("[%s] Unable to marshal %s collection for object IRI [%s]: %s",
			h.endpoint, h.refType, objectIRI, err)

		h.writeResponse(rw, http.StatusInternalServerError, nil)

		return
	}

	h.writeResponse(rw, http.StatusOK, activitiesCollBytes)
}

func (h *Activities) handleActivitiesPage(rw http.ResponseWriter, req *http.Request, objectIRI, id *url.URL) {
	var page *vocab.OrderedCollectionPageType

	var err error

	pageNum, ok := h.getPageNum(req)
	if ok {
		page, err = h.getPage(objectIRI, id,
			spi.WithPageSize(h.PageSize),
			spi.WithPageNum(pageNum),
			spi.WithSortOrder(spi.SortDescending),
		)
	} else {
		page, err = h.getPage(objectIRI, id,
			spi.WithPageSize(h.PageSize),
			spi.WithSortOrder(spi.SortDescending),
		)
	}

	if err != nil {
		logger.Errorf("[%s] Error retrieving page for object IRI [%s]: %s",
			h.endpoint, objectIRI, err)

		h.writeResponse(rw, http.StatusInternalServerError, nil)

		return
	}

	pageBytes, err := h.marshal(page)
	if err != nil {
		logger.Errorf("[%s] Unable to marshal page for object IRI [%s]: %s",
			h.endpoint, objectIRI, err)

		h.writeResponse(rw, http.StatusInternalServerError, nil)

		return
	}

	h.writeResponse(rw, http.StatusOK, pageBytes)
}

//nolint:dupl
func (h *Activities) getActivities(objectIRI, id *url.URL) (*vocab.OrderedCollectionType, error) {
	it, err := h.activityStore.QueryReferences(h.refType,
		spi.NewCriteria(
			spi.WithObjectIRI(objectIRI),
		),
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = it.Close()
		if err != nil {
			logger.Errorf("failed to close iterator: %s", err.Error())
		}
	}()

	firstURL, err := h.getPageURL(id, -1)
	if err != nil {
		return nil, err
	}

	lastURL, err := h.getPageURL(id, getLastPageNum(it.TotalItems(), h.PageSize, spi.SortDescending))
	if err != nil {
		return nil, err
	}

	return vocab.NewOrderedCollection(nil,
		vocab.WithContext(vocab.ContextActivityStreams),
		vocab.WithID(id),
		vocab.WithFirst(firstURL),
		vocab.WithLast(lastURL),
		vocab.WithTotalItems(it.TotalItems()),
	), nil
}

func (h *Activities) getPage(objectIRI, id *url.URL, opts ...spi.QueryOpt) (*vocab.OrderedCollectionPageType, error) {
	it, err := h.activityStore.QueryActivities(
		spi.NewCriteria(
			spi.WithReferenceType(h.refType),
			spi.WithObjectIRI(objectIRI),
		), opts...,
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = it.Close()
		if err != nil {
			logger.Errorf("failed to close iterator: %s", err.Error())
		}
	}()

	options := storeutil.GetQueryOptions(opts...)

	activities, err := storeutil.ReadActivities(it, options.PageSize)
	if err != nil {
		return nil, err
	}

	items := make([]*vocab.ObjectProperty, len(activities))

	for i, activity := range activities {
		items[i] = vocab.NewObjectProperty(vocab.WithActivity(activity))
	}

	id, prev, next, err := h.getIDPrevNextURL(id, it.TotalItems(), options)
	if err != nil {
		return nil, err
	}

	return vocab.NewOrderedCollectionPage(items,
		vocab.WithContext(vocab.ContextActivityStreams),
		vocab.WithID(id),
		vocab.WithPrev(prev),
		vocab.WithNext(next),
		vocab.WithTotalItems(it.TotalItems()),
	), nil
}

// Activity implements a REST handler that retrieves a single activity by ID.
type Activity struct {
	*handler
}

func (h *Activity) handle(w http.ResponseWriter, req *http.Request) {
	activityIRI, err := h.getActivityIRI(req)
	if err != nil {
		logger.Debugf("[%s] Get activity IRI: %s", h.endpoint, err)

		h.writeResponse(w, http.StatusBadRequest, nil)

		return
	}

	activity, err := h.activityStore.GetActivity(activityIRI)
	if err != nil {
		if errors.Is(err, spi.ErrNotFound) {
			logger.Debugf("[%s] Activity ID not found [%s]", h.endpoint, activityIRI)

			h.writeResponse(w, http.StatusNotFound, nil)

			return
		}

		logger.Errorf("[%s] Unable to retrieve activity [%s]: %s", h.endpoint, activityIRI, err)

		h.writeResponse(w, http.StatusInternalServerError, nil)

		return
	}

	activityBytes, err := h.marshal(activity)
	if err != nil {
		logger.Errorf("[%s] Unable to marshal activity [%s]: %s", h.endpoint, activityIRI, err)

		h.writeResponse(w, http.StatusInternalServerError, nil)

		return
	}

	logger.Debugf("[%s] Returning activity: %s", h.endpoint, activityBytes)

	h.writeResponse(w, http.StatusOK, activityBytes)
}

func (h *Activity) getActivityIRI(req *http.Request) (*url.URL, error) {
	id := getIDParam(req)

	if id == "" {
		return nil, errors.New("activity ID not specified")
	}

	activityID := fmt.Sprintf("%s/activities/%s", h.ObjectIRI, id)

	logger.Debugf("[%s] Retrieving activity from store [%s]", h.endpoint, activityID)

	activityIRI, err := url.Parse(activityID)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID [%s]: %w", id, err)
	}

	return activityIRI, nil
}
