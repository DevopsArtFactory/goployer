/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type DynamoDBClient struct {
	Client *dynamodb.Client
}

func NewDynamoDBClient(cfg aws.Config) DynamoDBClient {
	return DynamoDBClient{
		Client: dynamodb.NewFromConfig(cfg),
	}
}

func (d DynamoDBClient) CheckTableExists(tableName string) (bool, error) {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	result, err := d.Client.DescribeTable(context.Background(), input)
	if err != nil {
		var notFound *dbtypes.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return false, nil
		}
		fmt.Println(err.Error())
		return false, err
	}

	return result.Table != nil, nil
}

func (d DynamoDBClient) CreateTable(tableName string) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []dbtypes.AttributeDefinition{
			{
				AttributeName: aws.String(constants.HashKey),
				AttributeType: dbtypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []dbtypes.KeySchemaElement{
			{
				AttributeName: aws.String(constants.HashKey),
				KeyType:       dbtypes.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &dbtypes.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(constants.DefaultWriteThroughput),
			WriteCapacityUnits: aws.Int64(constants.DefaultReadThroughput),
		},
		TableName: aws.String(tableName),
	}

	_, err := d.Client.CreateTable(context.Background(), input)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func (d DynamoDBClient) MakeRecord(stack, config, tags string, asg string, tableName string, status, timezone string, additionalFields map[string]string) error {
	item := map[string]dbtypes.AttributeValue{
		"identifier":        &dbtypes.AttributeValueMemberS{Value: asg},
		"deployment_status": &dbtypes.AttributeValueMemberS{Value: status},
		"stack":             &dbtypes.AttributeValueMemberS{Value: stack},
		"config":            &dbtypes.AttributeValueMemberS{Value: config},
		"start_date":        &dbtypes.AttributeValueMemberS{Value: tool.GetBaseTimeWithTimezone(timezone).Format(time.RFC3339)},
		"tag":               &dbtypes.AttributeValueMemberS{Value: tags},
	}

	for k, v := range additionalFields {
		item[k] = &dbtypes.AttributeValueMemberS{Value: v}
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(tableName),
	}

	_, err := d.Client.PutItem(context.Background(), input)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	Logger.Debugf("deployment metric is saved")

	return nil
}

func (d DynamoDBClient) UpdateRecord(updateKey, asg string, tableName string, status, timezone string, updateFields map[string]interface{}) error {
	baseEx := "SET #S = :status, #T = :timestamp"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#S": updateKey,
			"#T": constants.StatusTimeStampKey[status],
		},
		ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
			":status":    &dbtypes.AttributeValueMemberS{Value: status},
			":timestamp": &dbtypes.AttributeValueMemberS{Value: tool.GetBaseTimeWithTimezone(timezone).Format(time.RFC3339)},
		},
		Key: map[string]dbtypes.AttributeValue{
			constants.HashKey: &dbtypes.AttributeValueMemberS{Value: asg},
		},
		TableName:        aws.String(tableName),
		UpdateExpression: aws.String(baseEx),
	}

	if updateFields != nil {
		ex := baseEx
		for k, v := range updateFields {
			ex = fmt.Sprintf("%s, #%s = :%s", ex, k, k)
			input.UpdateExpression = aws.String(ex)
			input.ExpressionAttributeNames[fmt.Sprintf("#%s", k)] = k
			if k != "requestSum" {
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dbtypes.AttributeValueMemberS{Value: v.(string)}
			} else {
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%f", v.(float64))}
			}
		}
	}

	_, err := d.Client.UpdateItem(context.Background(), input)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	Logger.Debugf("Status is updated to %s", status)

	return nil
}

// GetSingleItem retrieves single item for single autoscaling group
func (d DynamoDBClient) GetSingleItem(asg, tableName string) (map[string]dbtypes.AttributeValue, error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]dbtypes.AttributeValue{
			constants.HashKey: &dbtypes.AttributeValueMemberS{Value: asg},
		},
		TableName: aws.String(tableName),
	}

	result, err := d.Client.GetItem(context.Background(), input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return result.Item, err
}

// UpdateStatistics updates the status value on metric table
func (d DynamoDBClient) UpdateStatistics(asg string, tableName, timezone string, updateFields map[string]interface{}) error {
	baseEx := "SET #T = :statisticsRecordTime"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#T": "statistics_record_time",
		},
		ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
			":statisticsRecordTime": &dbtypes.AttributeValueMemberS{Value: tool.GetBaseTimeWithTimezone(timezone).Format(time.RFC3339)},
		},
		Key: map[string]dbtypes.AttributeValue{
			constants.HashKey: &dbtypes.AttributeValueMemberS{Value: asg},
		},
		TableName:        aws.String(tableName),
		UpdateExpression: aws.String(baseEx),
	}

	if updateFields != nil {
		ex := baseEx
		for k, v := range updateFields {
			ex = fmt.Sprintf("%s, #%s = :%s", ex, k, k)
			input.UpdateExpression = aws.String(ex)
			input.ExpressionAttributeNames[fmt.Sprintf("#%s", k)] = k
			if !tool.IsStringInArray(k, []string{"tg_request_count", "lb_request_count"}) {
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dbtypes.AttributeValueMemberS{Value: v.(string)}
			} else {
				statData := v.(map[string]map[string]float64)
				refined := map[string]dbtypes.AttributeValue{}
				for tg, vv := range statData {
					temp := map[string]dbtypes.AttributeValue{}
					for id, vvv := range vv {
						temp[id] = &dbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%f", vvv)}
					}
					refined[tg] = &dbtypes.AttributeValueMemberM{Value: temp}
				}
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dbtypes.AttributeValueMemberM{Value: refined}
			}
		}
	}

	_, err := d.Client.UpdateItem(context.Background(), input)
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
}
