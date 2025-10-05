# USP Msg flow

## Onboarding

Agent                    MTP Service                USP Service
  |                           |                          |
  |--- WebSocketConnect ----->|                          |
  |<-- WebSocketConnect ACK --|                          |
  |                           |                          |
  |-- OnBoard Notification -->|-- ProcessUSPMessage ---->|
  |                           |                          |
  |                           |<-- (Optional) GetSupportedProtocol --|
  |<-- GetSupportedProto -----|<-- SendMessageToAgent ---|
  |                           |                          |
  |--  GetSupportedProtoResp->|-- ProcessUSPMessage ---->|
  |                           |                          |
  |                           |<-- GetSupportedDM -------|
  |<--- GetSupportedDM -------|<-- SendMessageToAgent ---|
  |                           |                          |
  |--- GetSupportedDMResp --->|-- ProcessUSPMessage ---->|
