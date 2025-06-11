package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	TargetAttr          string
	TimeZone            string
	Namespace           string
	ScaledObject        string
}

func NewDynamoDBMetadata(scaledObject *pb.ScaledObjectRef) (*DynamoDBMetadata, error) {
	meta := &DynamoDBMetadata{
		TableName:           scaledObject.GetScalerMetadata()["table"],
		Region:              scaledObject.GetScalerMetadata()["region"],
		StartTimeAttr:       scaledObject.GetScalerMetadata()["startAttribute"],
		EndTimeAttr:         scaledObject.GetScalerMetadata()["endAttribute"],
		DesiredReplicasAttr: scaledObject.GetScalerMetadata()["desiredReplicasAttribute"],
		TargetAttr:          scaledObject.GetScalerMetadata()["targetAttribute"],
		TimeZone:            scaledObject.GetScalerMetadata()["timezone"],
		Namespace:           scaledObject.GetNamespace(),
		ScaledObject:        scaledObject.GetName(),
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
	if strings.TrimSpace(db.Meta.TargetAttr) == "" {
		return db.getEventsWithoutTargetAttr()
	} else {
		return db.getEventsWithTargetAttr()
	}
}

func (db *DynamoDBClient) getEventsWithoutTargetAttr() ([]Event, error) {
	location, err := time.LoadLocation(db.Meta.TimeZone)
	if err != nil {
		fmt.Printf("[DynamoDB Error] failed to load timezone '%s': %v\n", db.Meta.TimeZone, err)
		return nil, err
	}
	now := time.Now().In(location)
	nowStr := now.Format(time.RFC3339)
	filter := fmt.Sprintf("%s <= :now AND %s >= :now", db.Meta.StartTimeAttr, db.Meta.EndTimeAttr)
	exprAttrValues := map[string]types.AttributeValue{
		":now": &types.AttributeValueMemberS{Value: nowStr},
	}
	input := &dynamodb.ScanInput{
		TableName:                 &db.Meta.TableName,
		FilterExpression:          &filter,
		ExpressionAttributeValues: exprAttrValues,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := db.Client.Scan(ctx, input)
	if err != nil {
		fmt.Printf("[DynamoDB Error] failed to scan table '%s': %v\n", db.Meta.TableName, err)
		return nil, err
	}
	var events []Event
	for _, item := range result.Items {
		startStr := getStringAttr(item, db.Meta.StartTimeAttr)
		endStr := getStringAttr(item, db.Meta.EndTimeAttr)
		desiredReplicas := getIntAttr(item, db.Meta.DesiredReplicasAttr)
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			fmt.Printf("[DynamoDB Parse Error] failed to parse startStr '%s': %v\n", startStr, err)
			continue
		}
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			fmt.Printf("[DynamoDB Parse Error] failed to parse endStr '%s': %v\n", endStr, err)
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

func (db *DynamoDBClient) getEventsWithTargetAttr() ([]Event, error) {
	location, err := time.LoadLocation(db.Meta.TimeZone)
	if err != nil {
		fmt.Printf("[DynamoDB Error] failed to load timezone '%s': %v\n", db.Meta.TimeZone, err)
		return nil, err
	}
	now := time.Now().In(location)
	nowStr := now.Format(time.RFC3339)
	filter := fmt.Sprintf("%s <= :now AND %s >= :now", db.Meta.StartTimeAttr, db.Meta.EndTimeAttr)
	exprAttrValues := map[string]types.AttributeValue{
		":now": &types.AttributeValueMemberS{Value: nowStr},
	}
	input := &dynamodb.ScanInput{
		TableName:                 &db.Meta.TableName,
		FilterExpression:          &filter,
		ExpressionAttributeValues: exprAttrValues,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := db.Client.Scan(ctx, input)
	if err != nil {
		fmt.Printf("[DynamoDB Error] failed to scan table '%s': %v\n", db.Meta.TableName, err)
		return nil, err
	}
	var events []Event
	targetKey := db.Meta.Namespace + "/" + db.Meta.ScaledObject
	for _, item := range result.Items {
		startStr := getStringAttr(item, db.Meta.StartTimeAttr)
		endStr := getStringAttr(item, db.Meta.EndTimeAttr)
		desiredReplicas := getIntAttr(item, db.Meta.DesiredReplicasAttr)
		targets := getStringAttr(item, db.Meta.TargetAttr)
		addEvent := false
		for _, t := range strings.Split(targets, ",") {
			if strings.TrimSpace(t) == targetKey {
				addEvent = true
				break
			}
		}
		if !addEvent {
			continue
		}
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			fmt.Printf("[DynamoDB Parse Error] failed to parse startStr '%s': %v\n", startStr, err)
			continue
		}
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			fmt.Printf("[DynamoDB Parse Error] failed to parse endStr '%s': %v\n", endStr, err)
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
