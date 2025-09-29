package database

import (
	"time"

	"gorm.io/gorm"
)

// DeviceRepository handles device-related database operations
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create creates a new device
func (r *DeviceRepository) Create(device *Device) error {
	return r.db.Create(device).Error
}

// GetByID retrieves a device by ID
func (r *DeviceRepository) GetByID(id uint) (*Device, error) {
	var device Device
	err := r.db.Preload("Parameters").Preload("Alerts").Preload("Sessions").First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByEndpointID retrieves a device by endpoint ID
func (r *DeviceRepository) GetByEndpointID(endpointID string) (*Device, error) {
	var device Device
	err := r.db.Preload("Parameters").Preload("Alerts").Preload("Sessions").Where("endpoint_id = ?", endpointID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetAll retrieves all devices with pagination
func (r *DeviceRepository) GetAll(offset, limit int) ([]Device, error) {
	var devices []Device
	err := r.db.Offset(offset).Limit(limit).Find(&devices).Error
	return devices, err
}

// Update updates a device
func (r *DeviceRepository) Update(device *Device) error {
	return r.db.Save(device).Error
}

// Delete deletes a device (soft delete)
func (r *DeviceRepository) Delete(id uint) error {
	return r.db.Delete(&Device{}, id).Error
}

// UpdateLastSeen updates the last seen timestamp for a device
func (r *DeviceRepository) UpdateLastSeen(endpointID string) error {
	now := time.Now()
	return r.db.Model(&Device{}).Where("endpoint_id = ?", endpointID).Update("last_seen", now).Error
}

// GetOnlineDevices retrieves devices that are currently online
func (r *DeviceRepository) GetOnlineDevices() ([]Device, error) {
	var devices []Device
	err := r.db.Where("status = ?", "online").Find(&devices).Error
	return devices, err
}

// Count returns the total number of devices
func (r *DeviceRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&Device{}).Count(&count).Error
	return count, err
}

// ParameterRepository handles parameter-related database operations
type ParameterRepository struct {
	db *gorm.DB
}

// NewParameterRepository creates a new parameter repository
func NewParameterRepository(db *gorm.DB) *ParameterRepository {
	return &ParameterRepository{db: db}
}

// CreateOrUpdate creates or updates a parameter
func (r *ParameterRepository) CreateOrUpdate(parameter *Parameter) error {
	var existing Parameter
	err := r.db.Where("device_id = ? AND path = ?", parameter.DeviceID, parameter.Path).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new parameter
		parameter.LastUpdated = time.Now()
		return r.db.Create(parameter).Error
	} else if err != nil {
		return err
	}

	// Update existing parameter
	existing.Value = parameter.Value
	existing.Type = parameter.Type
	existing.Writable = parameter.Writable
	existing.LastUpdated = time.Now()
	return r.db.Save(&existing).Error
}

// GetByDeviceID retrieves all parameters for a device
func (r *ParameterRepository) GetByDeviceID(deviceID uint) ([]Parameter, error) {
	var parameters []Parameter
	err := r.db.Where("device_id = ?", deviceID).Find(&parameters).Error
	return parameters, err
}

// GetByPath retrieves parameters by path pattern
func (r *ParameterRepository) GetByPath(deviceID uint, pathPattern string) ([]Parameter, error) {
	var parameters []Parameter
	err := r.db.Where("device_id = ? AND path LIKE ?", deviceID, pathPattern).Find(&parameters).Error
	return parameters, err
}

// Delete deletes a parameter
func (r *ParameterRepository) Delete(deviceID uint, path string) error {
	return r.db.Where("device_id = ? AND path = ?", deviceID, path).Delete(&Parameter{}).Error
}

// AlertRepository handles alert-related database operations
type AlertRepository struct {
	db *gorm.DB
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// Create creates a new alert
func (r *AlertRepository) Create(alert *Alert) error {
	return r.db.Create(alert).Error
}

// GetByDeviceID retrieves alerts for a device
func (r *AlertRepository) GetByDeviceID(deviceID uint, resolved *bool) ([]Alert, error) {
	query := r.db.Where("device_id = ?", deviceID)
	if resolved != nil {
		query = query.Where("resolved = ?", *resolved)
	}

	var alerts []Alert
	err := query.Order("created_at DESC").Find(&alerts).Error
	return alerts, err
}

// GetAll retrieves all alerts with pagination
func (r *AlertRepository) GetAll(offset, limit int) ([]Alert, error) {
	var alerts []Alert
	err := r.db.Preload("Device").Offset(offset).Limit(limit).Order("created_at DESC").Find(&alerts).Error
	return alerts, err
}

// Resolve marks an alert as resolved
func (r *AlertRepository) Resolve(id uint) error {
	now := time.Now()
	return r.db.Model(&Alert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"resolved":    true,
		"resolved_at": now,
	}).Error
}

// SessionRepository handles session-related database operations
type SessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *Session) error {
	return r.db.Create(session).Error
}

// GetBySessionID retrieves a session by session ID
func (r *SessionRepository) GetBySessionID(sessionID string) (*Session, error) {
	var session Session
	err := r.db.Preload("Device").Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateActivity updates the last activity timestamp and increments message count
func (r *SessionRepository) UpdateActivity(sessionID string) error {
	now := time.Now()
	return r.db.Model(&Session{}).Where("session_id = ?", sessionID).Updates(map[string]interface{}{
		"last_activity": now,
		"message_count": gorm.Expr("message_count + 1"),
	}).Error
}

// Close closes a session
func (r *SessionRepository) Close(sessionID string) error {
	now := time.Now()
	return r.db.Model(&Session{}).Where("session_id = ?", sessionID).Updates(map[string]interface{}{
		"status":   "closed",
		"ended_at": now,
	}).Error
}

// GetActiveSessionsForDevice retrieves active sessions for a device
func (r *SessionRepository) GetActiveSessionsForDevice(deviceID uint) ([]Session, error) {
	var sessions []Session
	err := r.db.Where("device_id = ? AND status = ?", deviceID, "active").Find(&sessions).Error
	return sessions, err
}

// Repositories aggregates all repository instances
type Repositories struct {
	Device    *DeviceRepository
	Parameter *ParameterRepository
	Alert     *AlertRepository
	Session   *SessionRepository
}

// NewRepositories creates all repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Device:    NewDeviceRepository(db),
		Parameter: NewParameterRepository(db),
		Alert:     NewAlertRepository(db),
		Session:   NewSessionRepository(db),
	}
}
