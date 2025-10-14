package main

import (
	"encoding/hex"
	"log"

	v1_3 "openusp/pkg/proto/v1_3"
	v1_4 "openusp/pkg/proto/v1_4"

	"google.golang.org/protobuf/proto"
)

func main() {
	// Python-generated USP record (the exact bytes)
	// This is what the Python script sends:
	// Field 1: version = "1.3"
	// Field 2: to_id = "proto::openusp.controller"
	// Field 3: from_id = "os::012345-AAADF3FFCA7F"  
	// Field 10: stomp_connect with version = V1_2 (0)

	// Let's create the exact same record using Go
	record := &v1_3.Record{
		Version: "1.3",
		ToId:    "proto::openusp.controller",
		FromId:  "os::012345-AAADF3FFCA7F",
		RecordType: &v1_3.Record_StompConnect{
			StompConnect: &v1_3.STOMPConnectRecord{
				Version: v1_3.STOMPConnectRecord_V1_2,
			},
		},
	}

	// Marshal it
	recordBytes, err := proto.Marshal(record)
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}

	log.Printf("Created USP 1.3 StompConnect record: %d bytes", len(recordBytes))
	log.Printf("Hex: %s", hex.EncodeToString(recordBytes))

	// Now test if we can detect the version
	var testRecord14 v1_4.Record
	err14 := proto.Unmarshal(recordBytes, &testRecord14)
	log.Printf("\nTrying to unmarshal as USP 1.4: error=%v", err14)
	if err14 == nil {
		log.Printf("  Version field: '%s'", testRecord14.Version)
		log.Printf("  Match? %v", testRecord14.Version == "1.4")
	}

	var testRecord13 v1_3.Record
	err13 := proto.Unmarshal(recordBytes, &testRecord13)
	log.Printf("\nTrying to unmarshal as USP 1.3: error=%v", err13)
	if err13 == nil {
		log.Printf("  Version field: '%s'", testRecord13.Version)
		log.Printf("  Match? %v", testRecord13.Version == "1.3")
		log.Printf("  FromId: %s", testRecord13.FromId)
		log.Printf("  ToId: %s", testRecord13.ToId)
		if stomp := testRecord13.GetStompConnect(); stomp != nil {
			log.Printf("  StompConnect version: %v", stomp.Version)
		}
	}
}
