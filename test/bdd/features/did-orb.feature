#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@did-orb
Feature:
  Background: Setup
    Given variable "domain1IRI" is assigned the value "https://orb.domain1.com/services/orb"
    And variable "domain2IRI" is assigned the value "https://orb.domain2.com/services/orb"

    # domain2 server follows domain1 server
    Given variable "followID" is assigned a unique ID
    And variable "followActivity" is assigned the JSON value '{"@context":"https://www.w3.org/ns/activitystreams","id":"${domain2IRI}/activities/${followID}","type":"Follow","actor":"${domain2IRI}","to":"${domain1IRI}","object":"${domain1IRI}"}'
    When an HTTP POST is sent to "https://localhost:48426/services/orb/outbox" with content "${followActivity}" of type "application/json" signed with KMS key from "domain2"

    # domain1 server follows domain2 server
    Given variable "followID" is assigned a unique ID
    And variable "followActivity" is assigned the JSON value '{"@context":"https://www.w3.org/ns/activitystreams","id":"${domain1IRI}/activities/${followID}","type":"Follow","actor":"${domain1IRI}","to":"${domain2IRI}","object":"${domain2IRI}"}'
    When an HTTP POST is sent to "https://localhost:48326/services/orb/outbox" with content "${followActivity}" of type "application/json" signed with KMS key from "domain1"

    # domain1 invites domain2 to be a witness
    Given variable "inviteWitnessID" is assigned a unique ID
    And variable "inviteWitnessActivity" is assigned the JSON value '{"@context":["https://www.w3.org/ns/activitystreams","https://trustbloc.github.io/did-method-orb/contexts/anchor/v1"],"id":"${domain1IRI}/activities/${inviteWitnessID}","type":"InviteWitness","actor":"${domain1IRI}","to":"${domain2IRI}","object":"${domain2IRI}"}'
    When an HTTP POST is sent to "https://localhost:48326/services/orb/outbox" with content "${inviteWitnessActivity}" of type "application/json" signed with KMS key from "domain1"

    # domain2 invites domain1 to be a witness
    Given variable "inviteWitnessID" is assigned a unique ID
    And variable "inviteWitnessActivity" is assigned the JSON value '{"@context":["https://www.w3.org/ns/activitystreams","https://trustbloc.github.io/did-method-orb/contexts/anchor/v1"],"id":"${domain2IRI}/activities/${inviteWitnessID}","type":"InviteWitness","actor":"${domain2IRI}","to":"${domain1IRI}","object":"${domain1IRI}"}'
    When an HTTP POST is sent to "https://localhost:48426/services/orb/outbox" with content "${inviteWitnessActivity}" of type "application/json" signed with KMS key from "domain2"

    Then we wait 3 seconds

# TODO (#364): Re-enable these tests once DID documents have a domain that, that we can use via WebFinger to resolve from CAS.
#    @discover_did
#    Scenario: discover did
#      When client discover orb endpoints
#
#      Then container "orb-domain3" is stopped
#      Then we wait 3 seconds
#
#      # orb-domain1 keeps accepting requests
#      When client sends request to "https://orb.domain1.com/sidetree/v1/operations" to create DID document
#      Then check success response contains "#did"
#
#      Then we wait 2 seconds
#      When client sends request to "https://orb.domain1.com/sidetree/v1/identifiers" to resolve DID document
#      Then check success response contains "#did"
#      Then check success response contains "canonicalId"
#
#      When client sends request to "https://orb.domain1.com/sidetree/v1/identifiers" to resolve DID document with canonical id
#      Then check success response contains "#canonicalId"
#
#      When client sends request to "https://orb.domain1.com/sidetree/v1/operations" to recover DID document
#      Then check for request success
#      Then we wait 2 seconds
#
#      When client sends request to "https://orb.domain1.com/sidetree/v1/identifiers" to resolve DID document
#      Then check success response contains "recoveryKey"
#      Then check success response contains "canonicalId"
#
#      When client sends request to "https://orb.domain1.com/sidetree/v1/identifiers" to resolve DID document with canonical id
#      Then check success response contains "#canonicalId"
#      Then check success response contains "recoveryKey"
#
#      Then container "orb-domain3" is started
#      Then we wait 10 seconds
#
#      # resolve did in domain3 - it will trigger did discovery in different organisations
#      When client sends request to "https://orb.domain3.com/sidetree/v1/identifiers" to resolve DID document with canonical id
#      Then check error response contains "not found"
#
#      Then we wait 2 seconds
#      When client sends request to "https://orb.domain3.com/sidetree/v1/identifiers" to resolve DID document with canonical id
#      Then check success response contains "#canonicalId"
#      Then check success response contains "recoveryKey"

    @follow_anchor_writer_domain1
    Scenario: domain2 server follows domain1 server (anchor writer)

      # send create request to domain1
      When client sends request to "https://orb.domain1.com/sidetree/v1/operations" to create DID document
      Then check success response contains "#did"

      Then we wait 2 seconds
      When client sends request to "https://orb.domain1.com/sidetree/v1/identifiers" to resolve DID document
      Then check success response contains "#did"

      # check that document is available on the first server of domain2
      Then we wait 2 seconds
      When client sends request to "https://orb.domain2.com/sidetree/v1/identifiers" to resolve DID document
      Then check success response contains "#did"


    @follow_anchor_writer_domain2
    Scenario: domain1 server follows domain2 server (anchor writer)

      # send create request to domain2
      When client sends request to "https://orb.domain2.com/sidetree/v1/operations" to create DID document
      Then check success response contains "#did"

      Then we wait 2 seconds
      When client sends request to "https://orb.domain2.com/sidetree/v1/identifiers" to resolve DID document
      Then check success response contains "#did"

      # check that document is available on the first server of domain1
      Then we wait 2 seconds
      When client sends request to "https://orb.domain1.com/sidetree/v1/identifiers" to resolve DID document
      Then check success response contains "#did"

      # check that document is available on the second server of domain1
      When client sends request to "https://orb2.domain1.com/sidetree/v1/identifiers" to resolve DID document
      Then check success response contains "#did"

    @concurrent_requests_scenario
    Scenario: concurrent requests plus server shutdown tests

     # write batch of DIDs to multiple servers and check them
     When client sends request to "https://orb.domain1.com/sidetree/v1/operations,https://orb2.domain1.com/sidetree/v1/operations,https://orb.domain2.com/sidetree/v1/operations" to create 50 DID documents using 10 concurrent requests
     Then we wait 5 seconds
     Then client sends request to "https://orb.domain1.com/sidetree/v1/identifiers,https://orb2.domain1.com/sidetree/v1/identifiers,https://orb.domain2.com/sidetree/v1/identifiers" to verify the DID documents that were created

     Then container "orb-domain1" is stopped
     Then container "orb2-domain1" is stopped
     Then container "orb-domain2" is stopped
     Then container "ipfs" is stopped

     Then we wait 3 seconds

     Then container "orb-domain1" is started
     Then container "orb2-domain1" is started
     Then container "orb-domain2" is started
     Then container "ipfs" is started

     Then we wait 5 seconds

     # now that servers re-started we should check for DIDs that were created before shutdowns
     Then client sends request to "https://orb.domain1.com/sidetree/v1/identifiers,https://orb2.domain1.com/sidetree/v1/identifiers,https://orb.domain2.com/sidetree/v1/identifiers" to verify the DID documents that were created

     # write batch of DIDs to multiple servers again and check them
     When client sends request to "https://orb.domain1.com/sidetree/v1/operations,https://orb2.domain1.com/sidetree/v1/operations,https://orb.domain2.com/sidetree/v1/operations" to create 50 DID documents using 10 concurrent requests

     Then we wait 5 seconds
     Then client sends request to "https://orb.domain1.com/sidetree/v1/identifiers,https://orb2.domain1.com/sidetree/v1/identifiers,https://orb.domain2.com/sidetree/v1/identifiers" to verify the DID documents that were created
