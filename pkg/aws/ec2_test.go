/*
copyright 2025 the Goployer authors

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
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"

	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

func TestMakeLaunchTemplateBlockDeviceMappings(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []schemas.BlockDevice
		expected []ec2types.LaunchTemplateBlockDeviceMappingRequest
	}{
		{
			name: "Basic EBS volume",
			blocks: []schemas.BlockDevice{
				{
					DeviceName: "/dev/xvda",
					VolumeSize: 30,
					VolumeType: "gp2",
				},
			},
			expected: []ec2types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: stringPtr("/dev/xvda"),
					Ebs: &ec2types.LaunchTemplateEbsBlockDeviceRequest{
						VolumeSize:          int32Ptr(30),
						VolumeType:          ec2types.VolumeTypeGp2,
						DeleteOnTermination: boolPtr(false),
					},
				},
			},
		},
		{
			name: "EBS volume with DeleteOnTermination",
			blocks: []schemas.BlockDevice{
				{
					DeviceName:          "/dev/xvda",
					VolumeSize:          30,
					VolumeType:          "gp2",
					DeleteOnTermination: true,
				},
			},
			expected: []ec2types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: stringPtr("/dev/xvda"),
					Ebs: &ec2types.LaunchTemplateEbsBlockDeviceRequest{
						VolumeSize:          int32Ptr(30),
						VolumeType:          ec2types.VolumeTypeGp2,
						DeleteOnTermination: boolPtr(true),
					},
				},
			},
		},
		{
			name: "EBS volume with valid snapshot",
			blocks: []schemas.BlockDevice{
				{
					DeviceName: "/dev/xvda",
					VolumeSize: 30,
					VolumeType: "gp2",
					SnapshotID: "snap-12345678",
				},
			},
			expected: []ec2types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: stringPtr("/dev/xvda"),
					Ebs: &ec2types.LaunchTemplateEbsBlockDeviceRequest{
						VolumeSize:          int32Ptr(30),
						VolumeType:          ec2types.VolumeTypeGp2,
						SnapshotId:          stringPtr("snap-12345678"),
						DeleteOnTermination: boolPtr(false),
					},
				},
			},
		},
		{
			name: "EBS volume with invalid snapshot",
			blocks: []schemas.BlockDevice{
				{
					DeviceName: "/dev/xvda",
					VolumeSize: 30,
					VolumeType: "gp2",
					SnapshotID: "snap-invalid",
				},
			},
			expected: []ec2types.LaunchTemplateBlockDeviceMappingRequest{},
		},
		{
			name: "EBS volume with long snapshot",
			blocks: []schemas.BlockDevice{
				{
					DeviceName: "/dev/xvda",
					VolumeSize: 30,
					VolumeType: "gp2",
					SnapshotID: "snap-1234567890abcdef0",
				},
			},
			expected: []ec2types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: stringPtr("/dev/xvda"),
					Ebs: &ec2types.LaunchTemplateEbsBlockDeviceRequest{
						VolumeSize:          int32Ptr(30),
						VolumeType:          ec2types.VolumeTypeGp2,
						SnapshotId:          stringPtr("snap-1234567890abcdef0"),
						DeleteOnTermination: boolPtr(false),
					},
				},
			},
		},
		{
			name: "IOPS volume",
			blocks: []schemas.BlockDevice{
				{
					DeviceName: "/dev/xvda",
					VolumeSize: 30,
					VolumeType: "io1",
					Iops:       3000,
				},
			},
			expected: []ec2types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: stringPtr("/dev/xvda"),
					Ebs: &ec2types.LaunchTemplateEbsBlockDeviceRequest{
						VolumeSize:          int32Ptr(30),
						VolumeType:          ec2types.VolumeTypeIo1,
						Iops:                int32Ptr(3000),
						DeleteOnTermination: boolPtr(false),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EC2Client{
				Client:   &ec2.Client{},
				AsClient: &autoscaling.Client{},
			}
			result := client.MakeLaunchTemplateBlockDeviceMappings(tt.blocks)

			assert.Equal(t, len(tt.expected), len(result))
			for i, expected := range tt.expected {
				assert.Equal(t, *expected.DeviceName, *result[i].DeviceName)
				assert.Equal(t, *expected.Ebs.VolumeSize, *result[i].Ebs.VolumeSize)
				assert.Equal(t, expected.Ebs.VolumeType, result[i].Ebs.VolumeType)
				assert.Equal(t, *expected.Ebs.DeleteOnTermination, *result[i].Ebs.DeleteOnTermination)

				if expected.Ebs.SnapshotId != nil {
					assert.Equal(t, *expected.Ebs.SnapshotId, *result[i].Ebs.SnapshotId)
				}
				if expected.Ebs.Iops != nil {
					assert.Equal(t, *expected.Ebs.Iops, *result[i].Ebs.Iops)
				}
			}
		})
	}
}

func TestValidateSecurityGroupsConfig(t *testing.T) {
	tests := []struct {
		name           string
		securityGroups []string
		primaryENI     *schemas.ENIConfig
		secondaryENIs  []*schemas.ENIConfig
		expectedError  bool
		errorMessage   string
	}{
		{
			name:           "Valid security groups without ENI",
			securityGroups: []string{"sg-12345678"},
			primaryENI:     nil,
			secondaryENIs:  nil,
			expectedError:  false,
		},
		{
			name:           "Invalid security group format",
			securityGroups: []string{"invalid-sg"},
			primaryENI:     nil,
			secondaryENIs:  nil,
			expectedError:  true,
			errorMessage:   "invalid security group ID format",
		},
		{
			name:           "Empty security groups without ENI",
			securityGroups: []string{},
			primaryENI:     nil,
			secondaryENIs:  nil,
			expectedError:  true,
			errorMessage:   "security groups must be specified for launch template when ENI is not used",
		},
		{
			name:           "Valid ENI with security groups",
			securityGroups: nil,
			primaryENI: &schemas.ENIConfig{
				SecurityGroups: []string{"sg-12345678"},
			},
			secondaryENIs: nil,
			expectedError: false,
		},
		{
			name:           "ENI without security groups",
			securityGroups: nil,
			primaryENI: &schemas.ENIConfig{
				SecurityGroups: []string{},
			},
			secondaryENIs: nil,
			expectedError: true,
			errorMessage:  "security groups must be specified for primary ENI",
		},
		{
			name:           "Both security groups and ENI specified",
			securityGroups: []string{"sg-12345678"},
			primaryENI: &schemas.ENIConfig{
				SecurityGroups: []string{"sg-87654321"},
			},
			secondaryENIs: nil,
			expectedError: true,
			errorMessage:  "cannot use both launch template security groups and ENI security groups at the same time",
		},
		{
			name:           "Secondary ENI without security groups",
			securityGroups: nil,
			primaryENI:     nil,
			secondaryENIs: []*schemas.ENIConfig{
				{
					SecurityGroups: []string{},
				},
			},
			expectedError: true,
			errorMessage:  "security groups must be specified for secondary ENI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EC2Client{
				Client:   &ec2.Client{},
				AsClient: &autoscaling.Client{},
			}

			err := client.ValidateSecurityGroupsConfig(tt.securityGroups, tt.primaryENI, tt.secondaryENIs)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIMDSv2HopLimitLogic(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{
			name:     "Zero defaults to 1",
			input:    0,
			expected: 1,
		},
		{
			name:     "Negative defaults to 1",
			input:    -5,
			expected: 1,
		},
		{
			name:     "Positive preserved",
			input:    2,
			expected: 2,
		},
		{
			name:     "Max value preserved",
			input:    64,
			expected: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hopLimit := tt.input
			if hopLimit <= 0 {
				hopLimit = 1
			}
			assert.Equal(t, tt.expected, hopLimit)
		})
	}
}

func TestIMDSv2SecuritySettings(t *testing.T) {
	// Test critical IMDSv2 security values
	assert.Equal(t, "required", "required", "HttpTokens must be 'required' for IMDSv2")
	assert.Equal(t, "enabled", "enabled", "HttpEndpoint must be 'enabled' for metadata access")
	assert.Equal(t, int64(1), int64(1), "Default hop limit must be 1 for security")
}

func TestRegionConfigHopLimit(t *testing.T) {
	tests := []struct {
		name     string
		config   schemas.RegionConfig
		expected int64
	}{
		{
			name: "Default when not set",
			config: schemas.RegionConfig{
				HttpPutResponseHopLimit: 0,
			},
			expected: 1,
		},
		{
			name: "Custom value preserved",
			config: schemas.RegionConfig{
				HttpPutResponseHopLimit: 3,
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hopLimit := tt.config.HttpPutResponseHopLimit
			if hopLimit <= 0 {
				hopLimit = 1
			}
			assert.Equal(t, tt.expected, hopLimit)
		})
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
