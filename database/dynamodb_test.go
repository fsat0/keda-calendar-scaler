package database

import (
	"testing"
	pb "calendar-scaler/externalscaler"
)

func TestNewDynamoDBMetadata_RequiredFields(t *testing.T) {
	scaledObject := &pb.ScaledObjectRef{
		ScalerMetadata: map[string]string{
			"table": "calendar_events",
			"region": "ap-northeast-1",
			"startAttribute": "startEvent",
			"endAttribute": "endEvent",
			"desiredReplicasAttribute": "desiredReplicas",
			"timezone": "Asia/Tokyo",
		},
	}
	meta, err := NewDynamoDBMetadata(scaledObject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.TableName != "calendar_events" {
		t.Errorf("expected table name to be 'calendar_events', got '%s'", meta.TableName)
	}
}

func TestDynamoDBMetadata_Validate(t *testing.T) {
	meta := &DynamoDBMetadata{
		TableName:           "",
		StartTimeAttr:       "",
		EndTimeAttr:         "",
		DesiredReplicasAttr: "",
		TimeZone:            "",
	}
	err := meta.validate()
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}
