package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	pb "calendar-scaler/externalscaler"
)

type DynamoDBMetadata struct {
	TableName           string
	Region              string
	StartTimeAttr       string
	EndTimeAttr         string
	DesiredReplicasAttr string
	TimeZone            string
}

func NewDynamoDBMetadata(scaledObject *pb.ScaledObjectRef) (*DynamoDBMetadata, error) {
	meta := &DynamoDBMetadata{
		TableName:           scaledObject.GetScalerMetadata()["table"],
		Region:              scaledObject.GetScalerMetadata()["region"],
		StartTimeAttr:       scaledObject.GetScalerMetadata()["startAttribute"],
		EndTimeAttr:         scaledObject.GetScalerMetadata()["endAttribute"],
		DesiredReplicasAttr: scaledObject.GetScalerMetadata()["desiredReplicasAttribute"],
		TimeZone:            scaledObject.GetScalerMetadata()["timezone"],
	}
	if err := meta.validate(); err != nil {
		return nil, err
	}
	return meta, nil
}

func (meta *DynamoDBMetadata) validate() error {
	if meta.TableName == "" {
		return fmt.Errorf("table name is required")
	}
	if meta.StartTimeAttr == "" {
		return fmt.Errorf("startAttribute is required")
	}
	if meta.EndTimeAttr == "" {
		return fmt.Errorf("endAttribute is required")
	}
	if meta.DesiredReplicasAttr == "" {
		return fmt.Errorf("desiredReplicasAttribute is required")
	}
	if meta.TimeZone == "" {
		return fmt.Errorf("timezone is required")
	}
	return nil
}

type DynamoDBClient struct {
	Client *dynamodb.Client
	Meta   *DynamoDBMetadata
}

func NewDynamoDB(meta *DynamoDBMetadata) (*DynamoDBClient, error) {
	var cfg aws.Config
	var err error
	if meta.Region != "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(meta.Region))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}
	if err != nil {
		return nil, err
	}

	// Override endpoint if DYNAMODB_ENDPOINT environment variable is set
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	var client *dynamodb.Client
	if endpoint != "" {
		client = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = &endpoint
		})
	} else {
		client = dynamodb.NewFromConfig(cfg)
	}
	return &DynamoDBClient{Client: client, Meta: meta}, nil
}

func (db *DynamoDBClient) GetEvents() ([]Event, error) {
	location, err := time.LoadLocation(db.Meta.TimeZone)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(location)
	nowStr := now.Format(time.RFC3339)

	// Retrieve events within the period using DynamoDB Scan
	filter := fmt.Sprintf("%s <= :now AND %s >= :now", db.Meta.StartTimeAttr, db.Meta.EndTimeAttr)
	input := &dynamodb.ScanInput{
		TableName:        &db.Meta.TableName,
		FilterExpression: &filter,
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberS{Value: nowStr},
		},
	}
	result, err := db.Client.Scan(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	var events []Event
	for _, item := range result.Items {
		startStr := getStringAttr(item, db.Meta.StartTimeAttr)
		endStr := getStringAttr(item, db.Meta.EndTimeAttr)
		desiredReplicas := getIntAttr(item, db.Meta.DesiredReplicasAttr)
		start, err1 := time.Parse(time.RFC3339, startStr)
		end, err2 := time.Parse(time.RFC3339, endStr)
		if err1 != nil || err2 != nil {
			continue
		}
		events = append(events, Event{
			StartTime:       start,
			EndTime:         end,
			DesiredReplicas: desiredReplicas,
		})
	}
	return events, nil
}

func getStringAttr(item map[string]types.AttributeValue, key string) string {
	if v, ok := item[key]; ok {
		if s, ok := v.(*types.AttributeValueMemberS); ok {
			return s.Value
		}
	}
	return ""
}

func getIntAttr(item map[string]types.AttributeValue, key string) int {
	if v, ok := item[key]; ok {
		if n, ok := v.(*types.AttributeValueMemberN); ok {
			val, err := strconv.Atoi(n.Value)
			if err == nil {
				return val
			}
		}
	}
	return 0
}

func (db *DynamoDBClient) Close() error {
	// No explicit Close required for DynamoDB
	return nil
}
