package cwmp

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"

	"openusp/internal/tr181"
)

// MessageProcessor handles CWMP message processing
type MessageProcessor struct {
	tr181Manager      *tr181.DeviceManager
	onboardingManager *OnboardingManager
}

// Session represents a CWMP session
type Session struct {
	ID         string
	RemoteAddr string
	CreatedAt  time.Time
	LastAccess time.Time
	Timeout    time.Duration
	State      SessionState
	mu         sync.RWMutex
}

// SessionState represents the state of a CWMP session
type SessionState int

const (
	SessionStateNew SessionState = iota
	SessionStateActive
	SessionStateClosing
	SessionStateClosed
)

// CWMP Message Types
type Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  *Header  `xml:"Header,omitempty"`
	Body    Body     `xml:"Body"`
}

type Header struct {
	ID string `xml:"http://schemas.xmlsoap.org/ws/2004/08/addressing To,omitempty"`
}

type Body struct {
	Inform                         *Inform                         `xml:"urn:dslforum-org:cwmp-1-2 Inform,omitempty"`
	InformResponse                 *InformResponse                 `xml:"urn:dslforum-org:cwmp-1-2 InformResponse,omitempty"`
	GetRPCMethods                  *GetRPCMethods                  `xml:"urn:dslforum-org:cwmp-1-2 GetRPCMethods,omitempty"`
	GetRPCMethodsResponse          *GetRPCMethodsResponse          `xml:"urn:dslforum-org:cwmp-1-2 GetRPCMethodsResponse,omitempty"`
	GetParameterNames              *GetParameterNames              `xml:"urn:dslforum-org:cwmp-1-2 GetParameterNames,omitempty"`
	GetParameterNamesResponse      *GetParameterNamesResponse      `xml:"urn:dslforum-org:cwmp-1-2 GetParameterNamesResponse,omitempty"`
	GetParameterValues             *GetParameterValues             `xml:"urn:dslforum-org:cwmp-1-2 GetParameterValues,omitempty"`
	GetParameterValuesResponse     *GetParameterValuesResponse     `xml:"urn:dslforum-org:cwmp-1-2 GetParameterValuesResponse,omitempty"`
	SetParameterValues             *SetParameterValues             `xml:"urn:dslforum-org:cwmp-1-2 SetParameterValues,omitempty"`
	SetParameterValuesResponse     *SetParameterValuesResponse     `xml:"urn:dslforum-org:cwmp-1-2 SetParameterValuesResponse,omitempty"`
	GetParameterAttributes         *GetParameterAttributes         `xml:"urn:dslforum-org:cwmp-1-2 GetParameterAttributes,omitempty"`
	GetParameterAttributesResponse *GetParameterAttributesResponse `xml:"urn:dslforum-org:cwmp-1-2 GetParameterAttributesResponse,omitempty"`
	SetParameterAttributes         *SetParameterAttributes         `xml:"urn:dslforum-org:cwmp-1-2 SetParameterAttributes,omitempty"`
	SetParameterAttributesResponse *SetParameterAttributesResponse `xml:"urn:dslforum-org:cwmp-1-2 SetParameterAttributesResponse,omitempty"`
	Fault                          *Fault                          `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty"`
}

// Inform message structure
type Inform struct {
	DeviceId      DeviceIdStruct         `xml:"DeviceId"`
	Event         []EventStruct          `xml:"Event>EventStruct"`
	MaxEnvelopes  int                    `xml:"MaxEnvelopes"`
	CurrentTime   string                 `xml:"CurrentTime"`
	RetryCount    int                    `xml:"RetryCount"`
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

type DeviceIdStruct struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
	Type  string `xml:"Type,attr"`
}

// InformResponse message structure
type InformResponse struct {
	MaxEnvelopes int `xml:"MaxEnvelopes"`
}

// GetRPCMethods message structure
type GetRPCMethods struct{}

type GetRPCMethodsResponse struct {
	MethodList []string `xml:"MethodList>string"`
}

// GetParameterNames message structure
type GetParameterNames struct {
	ParameterPath string `xml:"ParameterPath"`
	NextLevel     bool   `xml:"NextLevel"`
}

type GetParameterNamesResponse struct {
	ParameterList []ParameterInfoStruct `xml:"ParameterList>ParameterInfoStruct"`
}

type ParameterInfoStruct struct {
	Name     string `xml:"Name"`
	Writable bool   `xml:"Writable"`
}

// GetParameterValues message structure
type GetParameterValues struct {
	ParameterNames []string `xml:"ParameterNames>string"`
}

type GetParameterValuesResponse struct {
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

// SetParameterValues message structure
type SetParameterValues struct {
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
	ParameterKey  string                 `xml:"ParameterKey"`
}

type SetParameterValuesResponse struct {
	Status int `xml:"Status"`
}

// TR-069 v1.2 Parameter Attributes message structures
type GetParameterAttributes struct {
	ParameterNames []string `xml:"ParameterNames>string"`
}

type GetParameterAttributesResponse struct {
	ParameterList []ParameterAttributeStruct `xml:"ParameterList>ParameterAttributeStruct"`
}

type SetParameterAttributes struct {
	ParameterList []ParameterAttributeStruct `xml:"ParameterList>ParameterAttributeStruct"`
}

type SetParameterAttributesResponse struct {
	Status int `xml:"Status"`
}

// TR-069 v1.2 Enhanced Parameter Attribute Structure
type ParameterAttributeStruct struct {
	Name               string   `xml:"Name"`
	Notification       int      `xml:"Notification"`       // 0=off, 1=passive, 2=active
	AccessList         []string `xml:"AccessList>string"`  // TR-069 v1.2 access control
	NotificationChange bool     `xml:"NotificationChange"` // TR-069 v1.2 attribute change notification
	AccessListChange   bool     `xml:"AccessListChange"`   // TR-069 v1.2 access list change
}

// Fault message structure
type Fault struct {
	FaultCode   string       `xml:"faultcode"`
	FaultString string       `xml:"faultstring"`
	Detail      *FaultDetail `xml:"detail,omitempty"`
}

type FaultDetail struct {
	CWMPFault *CWMPFault `xml:"urn:dslforum-org:cwmp-1-2 Fault,omitempty"`
}

type CWMPFault struct {
	FaultCode   int    `xml:"FaultCode"`
	FaultString string `xml:"FaultString"`
}

// NewMessageProcessor creates a new CWMP message processor
func NewMessageProcessor(tr181Manager *tr181.DeviceManager) *MessageProcessor {
	return &MessageProcessor{
		tr181Manager: tr181Manager,
	}
}

// SetOnboardingManager sets the onboarding manager for the processor
func (mp *MessageProcessor) SetOnboardingManager(onboardingManager *OnboardingManager) {
	mp.onboardingManager = onboardingManager
}

// ProcessRequest processes incoming CWMP requests
func (mp *MessageProcessor) ProcessRequest(session *Session, w http.ResponseWriter, r *http.Request) error {
	// Update session access time
	session.UpdateLastAccess()

	// Parse CWMP envelope
	envelope, err := mp.parseEnvelope(r)
	if err != nil {
		return fmt.Errorf("failed to parse CWMP envelope: %w", err)
	}

	// Process based on message type
	response, err := mp.processMessage(session, envelope)
	if err != nil {
		return fmt.Errorf("failed to process CWMP message: %w", err)
	}

	// Send response
	return mp.sendResponse(w, response)
}

// parseEnvelope parses CWMP SOAP envelope
func (mp *MessageProcessor) parseEnvelope(r *http.Request) (*Envelope, error) {
	var envelope Envelope
	decoder := xml.NewDecoder(r.Body)
	defer r.Body.Close()

	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("XML decode error: %w", err)
	}

	return &envelope, nil
}

// processMessage processes CWMP message and generates response
func (mp *MessageProcessor) processMessage(session *Session, envelope *Envelope) (*Envelope, error) {
	response := &Envelope{
		Body: Body{},
	}

	// Handle different message types
	if envelope.Body.Inform != nil {
		return mp.handleInform(session, envelope.Body.Inform)
	} else if envelope.Body.GetRPCMethods != nil {
		return mp.handleGetRPCMethods(session)
	} else if envelope.Body.GetParameterNames != nil {
		return mp.handleGetParameterNames(session, envelope.Body.GetParameterNames)
	} else if envelope.Body.GetParameterValues != nil {
		return mp.handleGetParameterValues(session, envelope.Body.GetParameterValues)
	} else if envelope.Body.SetParameterValues != nil {
		return mp.handleSetParameterValues(session, envelope.Body.SetParameterValues)
	}

	return response, nil
}

// handleInform handles Inform messages
func (mp *MessageProcessor) handleInform(session *Session, inform *Inform) (*Envelope, error) {
	// Log device information
	fmt.Printf("📱 Device Inform: %s %s (SN: %s)\n",
		inform.DeviceId.Manufacturer,
		inform.DeviceId.ProductClass,
		inform.DeviceId.SerialNumber)

	// Log events
	for _, event := range inform.Event {
		fmt.Printf("📋 Event: %s (Key: %s)\n", event.EventCode, event.CommandKey)
	}

	// Process device onboarding if onboarding manager is available
	if mp.onboardingManager != nil {
		go func() {
			// Run onboarding process in background
			sessionID := session.ID
			if _, err := mp.onboardingManager.ProcessInformForOnboarding(inform, sessionID); err != nil {
				fmt.Printf("⚠️  Device onboarding failed: %v\n", err)
			} else {
				fmt.Printf("✅ Device onboarding completed successfully\n")
			}
		}()
	}

	// Create InformResponse
	response := &Envelope{
		Body: Body{
			InformResponse: &InformResponse{
				MaxEnvelopes: 1,
			},
		},
	}

	return response, nil
}

// handleGetRPCMethods handles GetRPCMethods requests
func (mp *MessageProcessor) handleGetRPCMethods(session *Session) (*Envelope, error) {
	methods := []string{
		"GetRPCMethods",
		"GetParameterNames",
		"GetParameterValues",
		"SetParameterValues",
		"GetParameterAttributes",
		"SetParameterAttributes",
		"AddObject",
		"DeleteObject",
		"Reboot",
		"FactoryReset",
		// TR-069 v1.2 enhanced methods
		"Download",
		"Upload",
		"GetAllQueuedTransfers",
		"ScheduleInform",
		"SetVouchers",
		"GetOptions",
	}

	response := &Envelope{
		Body: Body{
			GetRPCMethodsResponse: &GetRPCMethodsResponse{
				MethodList: methods,
			},
		},
	}

	return response, nil
}

// handleGetParameterNames handles GetParameterNames requests
func (mp *MessageProcessor) handleGetParameterNames(session *Session, req *GetParameterNames) (*Envelope, error) {
	// Use TR-181 manager to get parameter names
	params, err := mp.tr181Manager.GetParameterNames(req.ParameterPath, req.NextLevel)
	if err != nil {
		return mp.createFaultResponse(9005, "Invalid parameter name"), nil
	}

	var paramList []ParameterInfoStruct
	for _, param := range params {
		paramList = append(paramList, ParameterInfoStruct{
			Name:     param.Name,
			Writable: param.Writable,
		})
	}

	response := &Envelope{
		Body: Body{
			GetParameterNamesResponse: &GetParameterNamesResponse{
				ParameterList: paramList,
			},
		},
	}

	return response, nil
}

// handleGetParameterValues handles GetParameterValues requests
func (mp *MessageProcessor) handleGetParameterValues(session *Session, req *GetParameterValues) (*Envelope, error) {
	var paramList []ParameterValueStruct

	for _, paramName := range req.ParameterNames {
		value, dataType, err := mp.tr181Manager.GetParameterValue(paramName)
		if err != nil {
			// Return fault for invalid parameter
			return mp.createFaultResponse(9005, fmt.Sprintf("Invalid parameter name: %s", paramName)), nil
		}

		paramList = append(paramList, ParameterValueStruct{
			Name:  paramName,
			Value: value,
			Type:  dataType,
		})
	}

	response := &Envelope{
		Body: Body{
			GetParameterValuesResponse: &GetParameterValuesResponse{
				ParameterList: paramList,
			},
		},
	}

	return response, nil
}

// handleSetParameterValues handles SetParameterValues requests
func (mp *MessageProcessor) handleSetParameterValues(session *Session, req *SetParameterValues) (*Envelope, error) {
	// Set parameters using TR-181 manager
	for _, param := range req.ParameterList {
		if err := mp.tr181Manager.SetParameterValue(param.Name, param.Value, param.Type); err != nil {
			// Return fault for invalid parameter or value
			return mp.createFaultResponse(9007, fmt.Sprintf("Invalid parameter value: %s", param.Name)), nil
		}
	}

	response := &Envelope{
		Body: Body{
			SetParameterValuesResponse: &SetParameterValuesResponse{
				Status: 0, // Success
			},
		},
	}

	return response, nil
}

// createFaultResponse creates a CWMP fault response
func (mp *MessageProcessor) createFaultResponse(faultCode int, faultString string) *Envelope {
	return &Envelope{
		Body: Body{
			Fault: &Fault{
				FaultCode:   "Client",
				FaultString: "CWMP fault",
				Detail: &FaultDetail{
					CWMPFault: &CWMPFault{
						FaultCode:   faultCode,
						FaultString: faultString,
					},
				},
			},
		},
	}
}

// sendResponse sends CWMP response
func (mp *MessageProcessor) sendResponse(w http.ResponseWriter, envelope *Envelope) error {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("SOAPAction", "")

	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	if err := encoder.Encode(envelope); err != nil {
		return fmt.Errorf("XML encode error: %w", err)
	}

	return nil
}

// NewSession creates a new CWMP session
func NewSession(id, remoteAddr string, timeout time.Duration) *Session {
	return &Session{
		ID:         id,
		RemoteAddr: remoteAddr,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		Timeout:    timeout,
		State:      SessionStateNew,
	}
}

// UpdateLastAccess updates the last access time
func (s *Session) UpdateLastAccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAccess = time.Now()
}

// IsExpired checks if session is expired
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastAccess) > s.Timeout
}

// Close closes the session
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = SessionStateClosed
}
