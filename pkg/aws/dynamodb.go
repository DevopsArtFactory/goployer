package aws

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	Logger "github.com/sirupsen/logrus"
	"time"
)

var (
	hashKey            = "identifier"
	statusTimeStampKey = map[string]string{
		"deployed":   "deployed_date_kst",
		"terminated": "terminated_date_kst",
	}
	DEFAULT_READ_THROUGHPUT  = int64(5)
	DEFAULT_WRITE_THROUGHPUT = int64(5)
)

type DynamoDBClient struct {
	Client *dynamodb.DynamoDB
}

func NewDynamoDBClient(session *session.Session, region string, creds *credentials.Credentials) DynamoDBClient {
	return DynamoDBClient{
		Client: getDynamoDBClientFn(session, region, creds),
	}
}

func getDynamoDBClientFn(session *session.Session, region string, creds *credentials.Credentials) *dynamodb.DynamoDB {
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
				AttributeName: aws.String(hashKey),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(hashKey),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(DEFAULT_WRITE_THROUGHPUT),
			WriteCapacityUnits: aws.Int64(DEFAULT_READ_THROUGHPUT),
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

func (d DynamoDBClient) MakeRecord(stack, config, tags string, asg string, tableName string, status string, additionalFields map[string]string) error {
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
			"start_date_kst": {
				S: aws.String(tool.GetKstTimestamp().Format(time.RFC3339)),
			},
			"tag": {
				S: aws.String(tags),
			},
		},
		TableName: aws.String(tableName),
	}

	if additionalFields != nil && len(additionalFields) > 0 {
		for k, v := range additionalFields {
			input.Item[k] = &dynamodb.AttributeValue{S: aws.String(v)}
		}
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

func (d DynamoDBClient) UpdateRecord(updateKey, asg string, tableName string, status string, updateFields map[string]string) error {
	baseEx := "SET #S = :status, #T = :timestamp"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#S": aws.String(updateKey),
			"#T": aws.String(statusTimeStampKey[status]),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(status),
			},
			":timestamp": {
				S: aws.String(tool.GetKstTimestamp().Format(time.RFC3339)),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			hashKey: {
				S: aws.String(asg),
			},
		},
		TableName:        aws.String(tableName),
		UpdateExpression: aws.String(baseEx),
	}

	if updateFields != nil && len(updateFields) > 0 {
		ex := baseEx
		for k, v := range updateFields {
			ex = fmt.Sprintf("%s, #%s = :%s", ex, k, k)
			input.UpdateExpression = aws.String(ex)
			input.ExpressionAttributeValues[fmt.Sprintf(":%s", k)] = &dynamodb.AttributeValue{
				S: aws.String(v),
			}
			input.ExpressionAttributeNames[fmt.Sprintf("#%s", k)] = aws.String(k)
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
			hashKey: {
				S: aws.String(asg),
			},
		},
		TableName: aws.String(tableName),
	}

	result, err := d.Client.GetItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
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
		return nil, err
	}

	return result.Item, err
}
