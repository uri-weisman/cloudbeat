// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package benchmark

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/elastic/cloudbeat/config"
	"github.com/elastic/cloudbeat/resources/fetching"
	"github.com/elastic/cloudbeat/resources/providers"
	"github.com/elastic/cloudbeat/resources/providers/awslib"
)

type expectedFetchers struct {
	names []string
	count int
}

func TestNewBenchmark(t *testing.T) {
	logger := logp.NewLogger("test new factory")

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		want    expectedFetchers
	}{
		{
			name: "Get k8s factory",
			cfg: &config.Config{
				Benchmark: config.CIS_K8S,
			},
			want: expectedFetchers{
				names: []string{
					fetching.FileSystemType,
					fetching.KubeAPIType,
					fetching.ProcessType,
				},
				count: 3,
			},
		},
		{
			name: "Get CIS AWS factory",
			cfg: &config.Config{
				Benchmark: config.CIS_AWS,
				CloudConfig: config.CloudConfig{
					AwsCred: aws.ConfigAWS{
						AccessKeyID: "test",
					},
				},
			},
			want: expectedFetchers{
				names: []string{
					fetching.IAMType,
					fetching.KmsType,
					fetching.TrailType,
					fetching.MonitoringType,
					fetching.EC2NetworkingType,
					fetching.RdsType,
					fetching.S3Type,
				},
				count: 7,
			},
		},
		{
			name: "Get CIS EKS factory without the aws related fetchers",
			cfg: &config.Config{
				Benchmark: config.CIS_EKS,
			},
			want: expectedFetchers{
				names: []string{
					fetching.FileSystemType,
					fetching.KubeAPIType,
					fetching.ProcessType,
				},
				count: 3,
			},
		},
		{
			name: "Get CIS EKS factory with aws related fetchers",
			cfg: &config.Config{
				Benchmark: config.CIS_EKS,
				CloudConfig: config.CloudConfig{
					AwsCred: aws.ConfigAWS{
						AccessKeyID: "test",
					},
				},
			},
			want: expectedFetchers{
				names: []string{
					fetching.FileSystemType,
					fetching.KubeAPIType,
					fetching.ProcessType,
					fetching.EcrType,
					fetching.ElbType,
				},
				count: 5,
			},
		},
		{
			name: "Non supported benchmark fail to get factory",
			cfg: &config.Config{
				Benchmark: "Non existing benchmark",
			},
			want: expectedFetchers{
				names: []string{},
				count: 0,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kubeClient := providers.MockKubernetesClientGetterAPI{}
			kubeClient.On("GetClient", mock.Anything, mock.Anything, mock.Anything).Return(k8sfake.NewSimpleClientset(), nil)

			awsCfg := &awslib.MockConfigProviderAPI{}
			awsCfg.EXPECT().InitializeAWSConfig(mock.Anything, mock.Anything).
				Call.
				Return(func(ctx context.Context, config aws.ConfigAWS) *awssdk.Config {
					return CreateSdkConfig(config, "us1-east")
				},
					func(ctx context.Context, config aws.ConfigAWS) error {
						return nil
					},
				)

			identityProvider := &awslib.MockIdentityProviderGetter{}
			identityProvider.EXPECT().GetIdentity(mock.Anything, mock.Anything).Return(&awslib.Identity{
				Account: awssdk.String("test-account"),
			}, nil)

			b, err := NewBenchmark(tt.cfg)
			if tt.wantErr {
				if b == nil {
					require.Error(t, err)
					return
				}
			} else {
				require.NoError(t, err)
			}
			fetchersMap, err := b.InitRegistry(
				context.Background(),
				logger,
				tt.cfg,
				make(chan fetching.ResourceInfo),
				NewDependencies(&kubeClient, identityProvider, awsCfg),
			)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.count, len(fetchersMap.Keys()))

			require.NoError(t, b.Run(context.Background()))
			defer b.Stop()
			for _, fetcher := range tt.want.names {
				ok := fetchersMap.ShouldRun(fetcher)
				assert.Truef(t, ok, "fetcher %s enabled", fetcher)
			}
		})
	}
}

func CreateSdkConfig(config aws.ConfigAWS, region string) *awssdk.Config {
	awsConfig := awssdk.NewConfig()
	awsCredentials := awssdk.Credentials{
		AccessKeyID:     config.AccessKeyID,
		SecretAccessKey: config.SecretAccessKey,
		SessionToken:    config.SessionToken,
	}

	awsConfig.Credentials = credentials.StaticCredentialsProvider{
		Value: awsCredentials,
	}
	awsConfig.Region = region
	return awsConfig
}