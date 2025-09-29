// Package usp provides USP record and message parsing for both v1.3 and v1.4
package usp

import (
	"encoding/base64"
	"fmt"
	"time"

	v1_3 "openusp/pkg/proto/v1_3"
	v1_4 "openusp/pkg/proto/v1_4"

	"google.golang.org/protobuf/proto"
)

// USPVersion represents the USP protocol version
type USPVersion string

const (
	Version13 USPVersion = "1.3"
	Version14 USPVersion = "1.4"
)

// USPRecord represents a parsed USP record with version info
type USPRecord struct {
	Version    USPVersion
	ToID       string
	FromID     string
	PayloadSec string
	Payload    []byte
	RecordType string
	Record13   *v1_3.Record // Only set if version is 1.3
	Record14   *v1_4.Record // Only set if version is 1.4
	RawData    []byte
}

// USPMessage represents a parsed USP message
type USPMessage struct {
	Version   USPVersion
	MsgID     string
	MsgType   string
	Body      interface{}
	Message13 *v1_3.Msg // Only set if version is 1.3
	Message14 *v1_4.Msg // Only set if version is 1.4
	RawData   []byte
}

// ParsedUSP contains both record and message information
type ParsedUSP struct {
	Record    *USPRecord
	Message   *USPMessage
	ParseTime time.Time
	Errors    []string
}

// USPParser handles parsing of USP records and messages
type USPParser struct {
	supportedVersions []USPVersion
}

// NewUSPParser creates a new USP parser
func NewUSPParser() *USPParser {
	return &USPParser{
		supportedVersions: []USPVersion{Version13, Version14},
	}
}

// DetectVersion detects the USP protocol version from raw record data
func (p *USPParser) DetectVersion(data []byte) (USPVersion, error) {
	// Try USP 1.4 first
	var record14 v1_4.Record
	if err := proto.Unmarshal(data, &record14); err == nil {
		if record14.Version == "1.4" {
			return Version14, nil
		}
	}

	// Try USP 1.3
	var record13 v1_3.Record
	if err := proto.Unmarshal(data, &record13); err == nil {
		if record13.Version == "1.3" {
			return Version13, nil
		}
	}

	return "", fmt.Errorf("unable to detect USP version from data")
}

// ParseRecord parses a USP record from raw data
func (p *USPParser) ParseRecord(data []byte) (*USPRecord, error) {
	version, err := p.DetectVersion(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect USP version: %w", err)
	}

	record := &USPRecord{
		Version: version,
		RawData: data,
	}

	switch version {
	case Version14:
		return p.parseRecord14(record)
	case Version13:
		return p.parseRecord13(record)
	default:
		return nil, fmt.Errorf("unsupported USP version: %s", version)
	}
}

// parseRecord14 parses a USP 1.4 record
func (p *USPParser) parseRecord14(record *USPRecord) (*USPRecord, error) {
	var uspRecord v1_4.Record
	if err := proto.Unmarshal(record.RawData, &uspRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.4 record: %w", err)
	}

	record.Record14 = &uspRecord
	record.ToID = uspRecord.ToId
	record.FromID = uspRecord.FromId

	// Extract payload based on record type
	switch recordType := uspRecord.RecordType.(type) {
	case *v1_4.Record_NoSessionContext:
		record.RecordType = "NoSessionContext"
		record.Payload = recordType.NoSessionContext.Payload
		record.PayloadSec = base64.StdEncoding.EncodeToString(record.Payload)
	case *v1_4.Record_SessionContext:
		record.RecordType = "SessionContext"
		// SessionContext payload is [][]byte, take first payload if available
		if len(recordType.SessionContext.Payload) > 0 {
			record.Payload = recordType.SessionContext.Payload[0]
			record.PayloadSec = base64.StdEncoding.EncodeToString(record.Payload)
		}
	case *v1_4.Record_WebsocketConnect:
		record.RecordType = "WebSocketConnect"
		// WebSocket connect doesn't have payload
	case *v1_4.Record_MqttConnect:
		record.RecordType = "MqttConnect"
		// MQTT connect doesn't have payload
	case *v1_4.Record_StompConnect:
		record.RecordType = "StompConnect"
		// STOMP connect doesn't have payload
	case *v1_4.Record_UdsConnect:
		record.RecordType = "UdsConnect"
		// UDS connect doesn't have payload
	default:
		return nil, fmt.Errorf("unknown USP 1.4 record type")
	}

	return record, nil
}

// parseRecord13 parses a USP 1.3 record
func (p *USPParser) parseRecord13(record *USPRecord) (*USPRecord, error) {
	var uspRecord v1_3.Record
	if err := proto.Unmarshal(record.RawData, &uspRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.3 record: %w", err)
	}

	record.Record13 = &uspRecord
	record.ToID = uspRecord.ToId
	record.FromID = uspRecord.FromId

	// Extract payload based on record type
	switch recordType := uspRecord.RecordType.(type) {
	case *v1_3.Record_NoSessionContext:
		record.RecordType = "NoSessionContext"
		record.Payload = recordType.NoSessionContext.Payload
		record.PayloadSec = base64.StdEncoding.EncodeToString(record.Payload)
	case *v1_3.Record_SessionContext:
		record.RecordType = "SessionContext"
		// SessionContext payload is [][]byte, take first payload if available
		if len(recordType.SessionContext.Payload) > 0 {
			record.Payload = recordType.SessionContext.Payload[0]
			record.PayloadSec = base64.StdEncoding.EncodeToString(record.Payload)
		}
	case *v1_3.Record_WebsocketConnect:
		record.RecordType = "WebSocketConnect"
		// WebSocket connect doesn't have payload
	case *v1_3.Record_MqttConnect:
		record.RecordType = "MqttConnect"
		// MQTT connect doesn't have payload
	case *v1_3.Record_StompConnect:
		record.RecordType = "StompConnect"
		// STOMP connect doesn't have payload
	case *v1_3.Record_UdsConnect:
		record.RecordType = "UdsConnect"
		// UDS connect doesn't have payload
	default:
		return nil, fmt.Errorf("unknown USP 1.3 record type")
	}

	return record, nil
}

// ParseMessage parses a USP message from payload data
func (p *USPParser) ParseMessage(payload []byte, version USPVersion) (*USPMessage, error) {
	message := &USPMessage{
		Version: version,
		RawData: payload,
	}

	switch version {
	case Version14:
		return p.parseMessage14(message)
	case Version13:
		return p.parseMessage13(message)
	default:
		return nil, fmt.Errorf("unsupported USP version: %s", version)
	}
}

// parseMessage14 parses a USP 1.4 message
func (p *USPParser) parseMessage14(message *USPMessage) (*USPMessage, error) {
	var uspMsg v1_4.Msg
	if err := proto.Unmarshal(message.RawData, &uspMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.4 message: %w", err)
	}

	message.Message14 = &uspMsg
	message.MsgID = uspMsg.Header.MsgId
	message.MsgType = uspMsg.Header.MsgType.String()

	// Parse message body based on type
	switch body := uspMsg.Body.MsgBody.(type) {
	case *v1_4.Body_Request:
		message.Body = p.parseRequest14(body.Request)
	case *v1_4.Body_Response:
		message.Body = p.parseResponse14(body.Response)
	case *v1_4.Body_Error:
		message.Body = p.parseError14(body.Error)
	default:
		return nil, fmt.Errorf("unknown USP 1.4 message body type")
	}

	return message, nil
}

// parseMessage13 parses a USP 1.3 message
func (p *USPParser) parseMessage13(message *USPMessage) (*USPMessage, error) {
	var uspMsg v1_3.Msg
	if err := proto.Unmarshal(message.RawData, &uspMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.3 message: %w", err)
	}

	message.Message13 = &uspMsg
	message.MsgID = uspMsg.Header.MsgId
	message.MsgType = uspMsg.Header.MsgType.String()

	// Parse message body based on type
	switch body := uspMsg.Body.MsgBody.(type) {
	case *v1_3.Body_Request:
		message.Body = p.parseRequest13(body.Request)
	case *v1_3.Body_Response:
		message.Body = p.parseResponse13(body.Response)
	case *v1_3.Body_Error:
		message.Body = p.parseError13(body.Error)
	default:
		return nil, fmt.Errorf("unknown USP 1.3 message body type")
	}

	return message, nil
}

// ParseUSP performs complete parsing of USP record and message
func (p *USPParser) ParseUSP(data []byte) (*ParsedUSP, error) {
	parseTime := time.Now()
	parsed := &ParsedUSP{
		ParseTime: parseTime,
		Errors:    []string{},
	}

	// Parse record
	record, err := p.ParseRecord(data)
	if err != nil {
		parsed.Errors = append(parsed.Errors, fmt.Sprintf("Record parsing error: %v", err))
		return parsed, err
	}
	parsed.Record = record

	// Parse message if payload exists
	if len(record.Payload) > 0 {
		message, err := p.ParseMessage(record.Payload, record.Version)
		if err != nil {
			parsed.Errors = append(parsed.Errors, fmt.Sprintf("Message parsing error: %v", err))
			// Continue even if message parsing fails
		} else {
			parsed.Message = message
		}
	}

	return parsed, nil
}

// Request parsing helpers for USP 1.4
func (p *USPParser) parseRequest14(req *v1_4.Request) map[string]interface{} {
	result := map[string]interface{}{
		"type": "request",
	}

	switch reqType := req.ReqType.(type) {
	case *v1_4.Request_Get:
		result["operation"] = "Get"
		result["param_paths"] = reqType.Get.ParamPaths
		result["max_depth"] = reqType.Get.MaxDepth
	case *v1_4.Request_Set:
		result["operation"] = "Set"
		result["allow_partial"] = reqType.Set.AllowPartial
		result["update_objs"] = len(reqType.Set.UpdateObjs)
	case *v1_4.Request_Add:
		result["operation"] = "Add"
		result["allow_partial"] = reqType.Add.AllowPartial
		result["create_objs"] = len(reqType.Add.CreateObjs)
	case *v1_4.Request_Delete:
		result["operation"] = "Delete"
		result["allow_partial"] = reqType.Delete.AllowPartial
		result["obj_paths"] = reqType.Delete.ObjPaths
	case *v1_4.Request_Operate:
		result["operation"] = "Operate"
		result["command"] = reqType.Operate.Command
		result["command_key"] = reqType.Operate.CommandKey
	case *v1_4.Request_Notify:
		result["operation"] = "Notify"
		result["subscription_id"] = reqType.Notify.SubscriptionId
	case *v1_4.Request_GetSupportedDm:
		result["operation"] = "GetSupportedDM"
		result["obj_paths"] = reqType.GetSupportedDm.ObjPaths
		result["first_level_only"] = reqType.GetSupportedDm.FirstLevelOnly
	case *v1_4.Request_GetInstances:
		result["operation"] = "GetInstances"
		result["obj_paths"] = reqType.GetInstances.ObjPaths
		result["first_level_only"] = reqType.GetInstances.FirstLevelOnly
	default:
		result["operation"] = "Unknown"
	}

	return result
}

// Request parsing helpers for USP 1.3
func (p *USPParser) parseRequest13(req *v1_3.Request) map[string]interface{} {
	result := map[string]interface{}{
		"type": "request",
	}

	switch reqType := req.ReqType.(type) {
	case *v1_3.Request_Get:
		result["operation"] = "Get"
		result["param_paths"] = reqType.Get.ParamPaths
		result["max_depth"] = reqType.Get.MaxDepth
	case *v1_3.Request_Set:
		result["operation"] = "Set"
		result["allow_partial"] = reqType.Set.AllowPartial
		result["update_objs"] = len(reqType.Set.UpdateObjs)
	case *v1_3.Request_Add:
		result["operation"] = "Add"
		result["allow_partial"] = reqType.Add.AllowPartial
		result["create_objs"] = len(reqType.Add.CreateObjs)
	case *v1_3.Request_Delete:
		result["operation"] = "Delete"
		result["allow_partial"] = reqType.Delete.AllowPartial
		result["obj_paths"] = reqType.Delete.ObjPaths
	case *v1_3.Request_Operate:
		result["operation"] = "Operate"
		result["command"] = reqType.Operate.Command
		result["command_key"] = reqType.Operate.CommandKey
	case *v1_3.Request_Notify:
		result["operation"] = "Notify"
		result["subscription_id"] = reqType.Notify.SubscriptionId
	case *v1_3.Request_GetSupportedDm:
		result["operation"] = "GetSupportedDM"
		result["obj_paths"] = reqType.GetSupportedDm.ObjPaths
		result["first_level_only"] = reqType.GetSupportedDm.FirstLevelOnly
	case *v1_3.Request_GetInstances:
		result["operation"] = "GetInstances"
		result["obj_paths"] = reqType.GetInstances.ObjPaths
		result["first_level_only"] = reqType.GetInstances.FirstLevelOnly
	default:
		result["operation"] = "Unknown"
	}

	return result
}

// Response parsing helpers for USP 1.4
func (p *USPParser) parseResponse14(resp *v1_4.Response) map[string]interface{} {
	result := map[string]interface{}{
		"type": "response",
	}

	switch respType := resp.RespType.(type) {
	case *v1_4.Response_GetResp:
		result["operation"] = "GetResp"
		result["req_path_results"] = len(respType.GetResp.ReqPathResults)
	case *v1_4.Response_SetResp:
		result["operation"] = "SetResp"
		result["updated_obj_results"] = len(respType.SetResp.UpdatedObjResults)
	case *v1_4.Response_AddResp:
		result["operation"] = "AddResp"
		result["created_obj_results"] = len(respType.AddResp.CreatedObjResults)
	case *v1_4.Response_DeleteResp:
		result["operation"] = "DeleteResp"
		result["deleted_obj_results"] = len(respType.DeleteResp.DeletedObjResults)
	case *v1_4.Response_OperateResp:
		result["operation"] = "OperateResp"
		result["operation_results"] = len(respType.OperateResp.OperationResults)
	case *v1_4.Response_NotifyResp:
		result["operation"] = "NotifyResp"
		result["subscription_id"] = respType.NotifyResp.SubscriptionId
	case *v1_4.Response_GetSupportedDmResp:
		result["operation"] = "GetSupportedDMResp"
		result["req_obj_results"] = len(respType.GetSupportedDmResp.ReqObjResults)
	case *v1_4.Response_GetInstancesResp:
		result["operation"] = "GetInstancesResp"
		result["req_path_results"] = len(respType.GetInstancesResp.ReqPathResults)
	default:
		result["operation"] = "Unknown"
	}

	return result
}

// Response parsing helpers for USP 1.3
func (p *USPParser) parseResponse13(resp *v1_3.Response) map[string]interface{} {
	result := map[string]interface{}{
		"type": "response",
	}

	switch respType := resp.RespType.(type) {
	case *v1_3.Response_GetResp:
		result["operation"] = "GetResp"
		result["req_path_results"] = len(respType.GetResp.ReqPathResults)
	case *v1_3.Response_SetResp:
		result["operation"] = "SetResp"
		result["updated_obj_results"] = len(respType.SetResp.UpdatedObjResults)
	case *v1_3.Response_AddResp:
		result["operation"] = "AddResp"
		result["created_obj_results"] = len(respType.AddResp.CreatedObjResults)
	case *v1_3.Response_DeleteResp:
		result["operation"] = "DeleteResp"
		result["deleted_obj_results"] = len(respType.DeleteResp.DeletedObjResults)
	case *v1_3.Response_OperateResp:
		result["operation"] = "OperateResp"
		result["operation_results"] = len(respType.OperateResp.OperationResults)
	case *v1_3.Response_NotifyResp:
		result["operation"] = "NotifyResp"
		result["subscription_id"] = respType.NotifyResp.SubscriptionId
	case *v1_3.Response_GetSupportedDmResp:
		result["operation"] = "GetSupportedDMResp"
		result["req_obj_results"] = len(respType.GetSupportedDmResp.ReqObjResults)
	case *v1_3.Response_GetInstancesResp:
		result["operation"] = "GetInstancesResp"
		result["req_path_results"] = len(respType.GetInstancesResp.ReqPathResults)
	default:
		result["operation"] = "Unknown"
	}

	return result
}

// Error parsing helpers
func (p *USPParser) parseError14(err *v1_4.Error) map[string]interface{} {
	return map[string]interface{}{
		"type":     "error",
		"err_code": err.ErrCode,
		"err_msg":  err.ErrMsg,
	}
}

func (p *USPParser) parseError13(err *v1_3.Error) map[string]interface{} {
	return map[string]interface{}{
		"type":     "error",
		"err_code": err.ErrCode,
		"err_msg":  err.ErrMsg,
	}
}

// GetSupportedVersions returns the list of supported USP versions
func (p *USPParser) GetSupportedVersions() []USPVersion {
	return p.supportedVersions
}

// ValidateRecord validates a USP record structure
func (p *USPParser) ValidateRecord(record *USPRecord) error {
	if record == nil {
		return fmt.Errorf("record is nil")
	}

	if record.ToID == "" {
		return fmt.Errorf("record missing ToID")
	}

	if record.FromID == "" {
		return fmt.Errorf("record missing FromID")
	}

	if record.Version != Version13 && record.Version != Version14 {
		return fmt.Errorf("unsupported USP version: %s", record.Version)
	}

	return nil
}

// ValidateMessage validates a USP message structure
func (p *USPParser) ValidateMessage(message *USPMessage) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	if message.MsgID == "" {
		return fmt.Errorf("message missing MsgID")
	}

	if message.MsgType == "" {
		return fmt.Errorf("message missing MsgType")
	}

	if message.Version != Version13 && message.Version != Version14 {
		return fmt.Errorf("unsupported USP version: %s", message.Version)
	}

	return nil
}
