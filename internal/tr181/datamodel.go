// Package tr181 provides TR-181 Device:2 data model support for USP
package tr181

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// DataModelDocument represents the TR-181 XML document structure
type DataModelDocument struct {
	XMLName   xml.Name   `xml:"document"`
	Spec      string     `xml:"spec,attr"`
	File      string     `xml:"file,attr"`
	DataTypes []DataType `xml:"dataType"`
	Models    []Model    `xml:"model"`
}

// Model represents a data model version
type Model struct {
	Name    string   `xml:"name,attr"`
	Objects []Object `xml:"object"`
}

// MaxEntriesType handles both integer and "unbounded" values
type MaxEntriesType struct {
	Value     int
	Unbounded bool
}

// UnmarshalXMLAttr implements xml.UnmarshalerAttr
func (m *MaxEntriesType) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value == "unbounded" {
		m.Unbounded = true
		m.Value = -1 // Use -1 to represent unbounded
		return nil
	}

	val, err := strconv.Atoi(attr.Value)
	if err != nil {
		return err
	}
	m.Value = val
	m.Unbounded = false
	return nil
}

// Object represents a TR-181 object
type Object struct {
	Name        string         `xml:"name,attr"`
	Access      string         `xml:"access,attr"`
	MinEntries  int            `xml:"minEntries,attr"`
	MaxEntries  MaxEntriesType `xml:"maxEntries,attr"`
	Version     string         `xml:"version,attr"`
	Description string         `xml:"description"`
	Parameters  []Parameter    `xml:"parameter"`
	Objects     []Object       `xml:"object"`
	Commands    []Command      `xml:"command"`
} // Parameter represents a TR-181 parameter
type Parameter struct {
	Name        string          `xml:"name,attr"`
	Access      string          `xml:"access,attr"`
	Version     string          `xml:"version,attr"`
	Status      string          `xml:"status,attr"`
	Description string          `xml:"description"`
	Syntax      ParameterSyntax `xml:"syntax"`
}

// ParameterSyntax defines the parameter type and constraints
type ParameterSyntax struct {
	String       *StringType   `xml:"string"`
	UnsignedInt  *UnsignedInt  `xml:"unsignedInt"`
	Int          *IntType      `xml:"int"`
	Boolean      *BooleanType  `xml:"boolean"`
	DateTime     *DateTimeType `xml:"dateTime"`
	Base64       *Base64Type   `xml:"base64"`
	Enumerations []Enumeration `xml:"enumeration"`
}

// StringType represents string parameter constraints
type StringType struct {
	Size        *Size         `xml:"size"`
	Pattern     *Pattern      `xml:"pattern"`
	Enumeration []Enumeration `xml:"enumeration"`
}

// Size represents string size constraints
type Size struct {
	MinLength int `xml:"minLength,attr"`
	MaxLength int `xml:"maxLength,attr"`
}

// Pattern represents string pattern constraint
type Pattern struct {
	Value string `xml:"value,attr"`
}

// Enumeration represents enumeration values
type Enumeration struct {
	Value       string `xml:"value,attr"`
	Description string `xml:"description"`
}

// UnsignedInt represents unsigned integer constraints
type UnsignedInt struct {
	Range *Range `xml:"range"`
}

// IntType represents integer constraints
type IntType struct {
	Range *Range `xml:"range"`
}

// Range represents number range constraints
type Range struct {
	MinInclusive int `xml:"minInclusive,attr"`
	MaxInclusive int `xml:"maxInclusive,attr"`
}

// BooleanType represents boolean type
type BooleanType struct{}

// DateTimeType represents datetime type
type DateTimeType struct{}

// Base64Type represents base64 type
type Base64Type struct{}

// Command represents a TR-181 command
type Command struct {
	Name        string     `xml:"name,attr"`
	Version     string     `xml:"version,attr"`
	Async       bool       `xml:"async,attr"`
	Description string     `xml:"description"`
	Input       *CommandIO `xml:"input"`
	Output      *CommandIO `xml:"output"`
}

// CommandIO represents command input/output parameters
type CommandIO struct {
	Parameters []Parameter `xml:"parameter"`
}

// Event represents a TR-181 event
type Event struct {
	Name        string      `xml:"name,attr"`
	Version     string      `xml:"version,attr"`
	Description string      `xml:"description"`
	Parameters  []Parameter `xml:"parameter"`
}

// DataType represents a custom data type definition
type DataType struct {
	Name        string `xml:"name,attr"`
	Base        string `xml:"base,attr"`
	Description string `xml:"description"`
}

// DeviceNamespace represents the TR-181 Device namespace for USP operations
type DeviceNamespace struct {
	dataModel *DataModelDocument
	objects   map[string]*Object
}

// NewDeviceNamespace creates a new Device namespace from TR-181 XML data
func NewDeviceNamespace(xmlData []byte) (*DeviceNamespace, error) {
	var doc DataModelDocument
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse TR-181 XML: %v", err)
	}

	ns := &DeviceNamespace{
		dataModel: &doc,
		objects:   make(map[string]*Object),
	}

	// Index all objects by path for quick lookup
	for _, model := range doc.Models {
		ns.indexObjects(model.Objects, "")
	}

	return ns, nil
}

// indexObjects recursively indexes all objects by their full path
func (ns *DeviceNamespace) indexObjects(objects []Object, parentPath string) {
	for i := range objects {
		obj := &objects[i]
		fullPath := parentPath + obj.Name
		ns.objects[fullPath] = obj

		// Recursively index child objects
		ns.indexObjects(obj.Objects, fullPath)
	}
}

// GetObject returns the object definition for a given path
func (ns *DeviceNamespace) GetObject(path string) (*Object, bool) {
	obj, exists := ns.objects[path]
	return obj, exists
}

// GetParameter returns the parameter definition for a given path
func (ns *DeviceNamespace) GetParameter(path string) (*Parameter, error) {
	// Split the path to get object path and parameter name
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return nil, fmt.Errorf("invalid parameter path: %s", path)
	}

	objectPath := path[:lastDot+1] // Include the trailing dot
	paramName := path[lastDot+1:]

	obj, exists := ns.GetObject(objectPath)
	if !exists {
		return nil, fmt.Errorf("object not found: %s", objectPath)
	}

	// Find the parameter in the object
	for i := range obj.Parameters {
		if obj.Parameters[i].Name == paramName {
			return &obj.Parameters[i], nil
		}
	}

	return nil, fmt.Errorf("parameter not found: %s", paramName)
}

// GetCommand returns the command definition for a given path
func (ns *DeviceNamespace) GetCommand(path string) (*Command, error) {
	// Split the path to get object path and command name
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return nil, fmt.Errorf("invalid command path: %s", path)
	}

	objectPath := path[:lastDot+1] // Include the trailing dot
	commandName := path[lastDot+1:]

	obj, exists := ns.GetObject(objectPath)
	if !exists {
		return nil, fmt.Errorf("object not found: %s", objectPath)
	}

	// Find the command in the object
	for i := range obj.Commands {
		if obj.Commands[i].Name == commandName {
			return &obj.Commands[i], nil
		}
	}

	return nil, fmt.Errorf("command not found: %s", commandName)
}

// ValidatePath validates if a path exists in the data model
func (ns *DeviceNamespace) ValidatePath(path string) bool {
	// Check if it's an object path
	if _, exists := ns.GetObject(path); exists {
		return true
	}

	// Check if it's a parameter path
	if _, err := ns.GetParameter(path); err == nil {
		return true
	}

	// Check if it's a command path
	if _, err := ns.GetCommand(path); err == nil {
		return true
	}

	return false
}

// GetSupportedObjectPaths returns all supported object paths
func (ns *DeviceNamespace) GetSupportedObjectPaths() []string {
	var paths []string
	for path := range ns.objects {
		paths = append(paths, path)
	}
	return paths
}

// GetObjectParameters returns all parameters for a given object path
func (ns *DeviceNamespace) GetObjectParameters(objectPath string) []Parameter {
	obj, exists := ns.GetObject(objectPath)
	if !exists {
		return nil
	}
	return obj.Parameters
}

// GetObjectCommands returns all commands for a given object path
func (ns *DeviceNamespace) GetObjectCommands(objectPath string) []Command {
	obj, exists := ns.GetObject(objectPath)
	if !exists {
		return nil
	}
	return obj.Commands
}

// IsWritableParameter checks if a parameter is writable
func (ns *DeviceNamespace) IsWritableParameter(path string) bool {
	param, err := ns.GetParameter(path)
	if err != nil {
		return false
	}
	return param.Access == "readWrite"
}

// GetParameterType returns the type of a parameter
func (ns *DeviceNamespace) GetParameterType(path string) (string, bool) {
	param, err := ns.GetParameter(path)
	if err != nil {
		return "", false
	}

	// Determine type from syntax
	syntax := param.Syntax
	if syntax.String != nil {
		return "string", true
	}
	if syntax.UnsignedInt != nil {
		return "unsignedInt", true
	}
	if syntax.Int != nil {
		return "int", true
	}
	if syntax.Boolean != nil {
		return "boolean", true
	}
	if syntax.DateTime != nil {
		return "dateTime", true
	}
	if syntax.Base64 != nil {
		return "base64", true
	}

	return "unknown", true
}

// GetObjectPaths returns object paths matching the given path pattern
func (ns *DeviceNamespace) GetObjectPaths(parameterPath string, nextLevel bool) []string {
	var paths []string

	// Handle root path
	if parameterPath == "" || parameterPath == "Device." {
		for path := range ns.objects {
			if strings.HasPrefix(path, "Device.") {
				if nextLevel {
					// Only return immediate children
					subPath := strings.TrimPrefix(path, "Device.")
					parts := strings.Split(subPath, ".")
					if len(parts) == 2 && parts[1] == "" { // Object ends with .
						paths = append(paths, path)
					}
				} else {
					paths = append(paths, path)
				}
			}
		}
		return paths
	}

	// Find matching paths
	for path := range ns.objects {
		if strings.HasPrefix(path, parameterPath) {
			if nextLevel {
				// Only return immediate children
				remainder := strings.TrimPrefix(path, parameterPath)
				if remainder != "" {
					parts := strings.Split(remainder, ".")
					if len(parts) <= 2 { // At most one level down
						paths = append(paths, path)
					}
				}
			} else {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

// IsParameterWritable checks if a parameter is writable
func (ns *DeviceNamespace) IsParameterWritable(path string) bool {
	param, err := ns.GetParameter(path)
	if err != nil {
		return false
	}

	return param.Access == "readWrite"
}
