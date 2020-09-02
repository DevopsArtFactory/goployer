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
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type DynamoDBClient struct {
	Client *dynamodb.DynamoDB
}

func NewDynamoDBClient(session client.ConfigProvider, region string, creds *credentials.Credentials) DynamoDBClient {
	return DynamoDBClient{
		Client: getDynamoDBClientFn(session, region, creds),
	}
}

func getDynamoDBClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *dynamodb.DynamoDB {
	if creds == nil {
		return dynamodb.New(session, &aws.Config{Region: aws.String(region)})
	}
	return dynamodb.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func (d DynamoDBClient) CheckTableExists(tableName string) (bool, error) {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	result, err := d.Client.DescribeTable(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
				return false, nil
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false, err
	}

	if result.Table == nil {
		return false, nil
	}

	return true, nil
}

func (d DynamoDBClient) CreateTable(tableName string) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(constants.HashKey),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(constants.HashKey),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(constants.DefaultWriteThroughput),
			WriteCapacityUnits: aws.Int64(constants.DefaultReadThroughput),
		},
		TableName: aws.String(tableName),
	}

	_, err := d.Client.CreateTable(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceInUseException:
				fmt.Println(dynamodb.ErrCodeResourceInUseException, aerr.Error())
			case dynamodb.ErrCodeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	return nil
}

func (d DynamoDBClient) MakeRecord(stack, config, tags string, asg string, tableName string, status, timezone string, additionalFields map[string]string) error {
	input := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"identifier": {
				S: aws.String(asg),
			},
			"deployment_status": {
				S: aws.String(status),
			},
			"stack": {
				S: aws.String(stack),
			},
			"config": {
				S: aws.String(config),
			},
			"start_date": {
				S: aws.String(tool.GetBaseTimeWithTimezone(timezone).Format(time.RFC3339)),
			},
			"tag": {
				S: aws.String(tags),
			},
		},
		TableName: aws.String(tableName),
	}

	for k, v := range additionalFields {
		input.Item[k] = &dynamodb.AttributeValue{S: aws.String(v)}
	}

	_, err := d.Client.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeTransactionConflictException:
				fmt.Println(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	Logger.Debugf("deployment metric is saved")

	return nil
}

func (d DynamoDBClient) UpdateRecord(updateKey, asg string, tableName string, status, timezone string, updateFields map[string]interface{}) error {
	baseEx := "SET #S = :status, #T = :timestamp"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#S": aws.String(updateKey),
			"#T": aws.String(constants.StatusTimeStampKey[status]),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(status),
			},
			":timestamp": {
				S: aws.String(tool.GetBaseTimeWithTimezone(timezone).Format(time.RFC3339)),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			constants.HashKey: {
				S: aws.String(asg),
			},
		},
		TableName:        aws.String(tableName),
		UpdateExpression: aws.String(baseEx),
	}

	if updateFields != nil {
		ex := baseEx
		for k, v := range updateFields {
			ex = fmt.Sprintf("%s, #%s = :%s", ex, k, k)
			input.UpdateExpression = aws.String(ex)
			input.ExpressionAttributeNames[fmt.Sprintf("#%s", k)] = aws.String(k)
			if k != "requestSum" {
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dynamodb.AttributeValue{
					S: aws.String(v.(string)),
				}
			} else {
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dynamodb.AttributeValue{
					N: aws.String(fmt.Sprintf("%f", v.(float64))),
				}
			}
		}
	}

	_, err := d.Client.UpdateItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeTransactionConflictException:
				fmt.Println(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	Logger.Debugf("Status is updated to %s", status)

	return nil
}

func (d DynamoDBClient) GetSingleItem(asg, tableName string) (map[string]*dynamodb.AttributeValue, error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			constants.HashKey: {
				S: aws.String(asg),
			},
		},
		TableName: aws.String(tableName),
	}

	result, err := d.Client.GetItem(input)
	if err != nil {
		return nil, err
	}

	return result.Item, err
}

// UpdateStatistics updates the status value on metric table
func (d DynamoDBClient) UpdateStatistics(asg string, tableName, timezone string, updateFields map[string]interface{}) error {
	baseEx := "SET #T = :statisticsRecordTime"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#T": aws.String("statistics_record_time"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":statisticsRecordTime": {
				S: aws.String(tool.GetBaseTimeWithTimezone(timezone).Format(time.RFC3339)),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			constants.HashKey: {
				S: aws.String(asg),
			},
		},
		TableName:        aws.String(tableName),
		UpdateExpression: aws.String(baseEx),
	}

	if updateFields != nil {
		ex := baseEx
		for k, v := range updateFields {
			ex = fmt.Sprintf("%s, #%s = :%s", ex, k, k)
			input.UpdateExpression = aws.String(ex)
			input.ExpressionAttributeNames[fmt.Sprintf("#%s", k)] = aws.String(k)
			if k != "stat" {
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dynamodb.AttributeValue{
					S: aws.String(v.(string)),
				}
			} else {
				// stat data
				statData := v.(map[string]map[string]float64)
				refined := map[string]*dynamodb.AttributeValue{}
				for tg, vv := range statData {
					temp := map[string]*dynamodb.AttributeValue{}
					for id, vvv := range vv {
						temp[id] = &dynamodb.AttributeValue{
							N: aws.String(fmt.Sprintf("%f", vvv)),
						}
					}
					refined[tg] = &dynamodb.AttributeValue{
						M: temp,
					}
				}
				input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dynamodb.AttributeValue{
					M: refined,
				}
			}
		}
	}

	_, err := d.Client.UpdateItem(input)
	if err != nil {
		return err
	}

	return nil
}
