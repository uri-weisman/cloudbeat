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

package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/googleapis/gax-go/v2"
	"github.com/samber/lo"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/cloudbeat/resources/fetching"
	"github.com/elastic/cloudbeat/resources/providers/gcplib/auth"
)

type Provider struct {
	log       *logp.Logger
	ctx       context.Context
	config    auth.GcpFactoryConfig
	inventory *AssetsInventoryWrapper
	crm       *ResourceManagerWrapper
	crmCache  map[string]*fetching.EcsGcp
}

type AssetsInventoryWrapper struct {
	Close      func() error
	ListAssets func(ctx context.Context, req *assetpb.ListAssetsRequest, opts ...gax.CallOption) Iterator
}

type ResourceManagerWrapper struct {
	// returns project display name or an empty string
	getProjectDisplayName func(ctx context.Context, parent string) string

	// returns org display name or an empty string
	getOrganizationDisplayName func(ctx context.Context, parent string) string
}

type MonitoringAsset struct {
	Ecs        *fetching.EcsGcp
	LogMetrics []*ExtendedGcpAsset `json:"log_metrics,omitempty"`
	Alerts     []*ExtendedGcpAsset `json:"alerts,omitempty"`
}

type LoggingAsset struct {
	Ecs      *fetching.EcsGcp
	LogSinks []*ExtendedGcpAsset `json:"log_sinks,omitempty"`
}

type ServiceUsageAsset struct {
	Ecs      *fetching.EcsGcp
	Services []*ExtendedGcpAsset `json:"services,omitempty"`
}

type ExtendedGcpAsset struct {
	*assetpb.Asset
	Ecs *fetching.EcsGcp
}

type ProviderInitializer struct{}

type GcpAssetIDs struct {
	orgId         string
	projectId     string
	parentProject string
	parentOrg     string
}

type dnsPolicyFields struct {
	networks      []string
	enableLogging bool
}

type Iterator interface {
	Next() (*assetpb.Asset, error)
}

type ServiceAPI interface {
	// ListAllAssetTypesByName List all content types of the given assets types
	ListAllAssetTypesByName(assets []string) ([]*ExtendedGcpAsset, error)

	// ListMonitoringAssets List all monitoring assets by project id
	ListMonitoringAssets(map[string][]string) ([]*MonitoringAsset, error)

	// ListLoggingAssets returns a list of logging assets grouped by project id, extended with folder and org level log sinks
	ListLoggingAssets() ([]*LoggingAsset, error)

	// ListServiceUsageAssets returns a list of service usage assets grouped by project id
	ListServiceUsageAssets() ([]*ServiceUsageAsset, error)

	// Close the GCP asset client
	Close() error
}

type ProviderInitializerAPI interface {
	// Init initializes the GCP asset client
	Init(ctx context.Context, log *logp.Logger, gcpConfig auth.GcpFactoryConfig) (ServiceAPI, error)
}

func (p *ProviderInitializer) Init(ctx context.Context, log *logp.Logger, gcpConfig auth.GcpFactoryConfig) (ServiceAPI, error) {
	// initialize GCP assets inventory client
	client, err := asset.NewClient(ctx, gcpConfig.ClientOpts...)
	if err != nil {
		return nil, err
	}
	// wrap the assets inventory client for mocking
	assetsInventoryWrapper := &AssetsInventoryWrapper{
		Close: client.Close,
		ListAssets: func(ctx context.Context, req *assetpb.ListAssetsRequest, opts ...gax.CallOption) Iterator {
			return client.ListAssets(ctx, req, opts...)
		},
	}

	// initialize GCP resource manager client
	var gcpClientOpt []option.ClientOption
	gcpClientOpt = append(append(gcpClientOpt, option.WithScopes(cloudresourcemanager.CloudPlatformReadOnlyScope)), gcpConfig.ClientOpts...)
	crmService, err := cloudresourcemanager.NewService(ctx, gcpClientOpt...)
	if err != nil {
		return nil, err
	}
	// wrap the resource manager client for mocking
	crmServiceWrapper := &ResourceManagerWrapper{
		getProjectDisplayName: func(ctx context.Context, parent string) string {
			prj, err := crmService.Projects.Get(parent).Context(ctx).Do()
			if err != nil {
				log.Errorf("error fetching GCP Project: %s", err)
				return ""
			}
			return prj.DisplayName
		},
		getOrganizationDisplayName: func(ctx context.Context, parent string) string {
			org, err := crmService.Organizations.Get(parent).Context(ctx).Do()
			if err != nil {
				log.Errorf("error fetching GCP Org: %s", err)
				return ""
			}
			return org.DisplayName
		},
	}

	return &Provider{
		config:    gcpConfig,
		log:       log,
		ctx:       ctx,
		inventory: assetsInventoryWrapper,
		crm:       crmServiceWrapper,
		crmCache:  make(map[string]*fetching.EcsGcp),
	}, nil
}

func (p *Provider) ListAllAssetTypesByName(assetTypes []string) ([]*ExtendedGcpAsset, error) {
	p.log.Infof("Listing GCP asset types: %v in %v", assetTypes, p.config.Parent)

	wg := sync.WaitGroup{}
	var resourceAssets []*assetpb.Asset
	var policyAssets []*assetpb.Asset

	wg.Add(1)
	go func() {
		request := &assetpb.ListAssetsRequest{
			Parent:      p.config.Parent,
			AssetTypes:  assetTypes,
			ContentType: assetpb.ContentType_RESOURCE,
		}
		resourceAssets = getAllAssets(p.log, p.inventory.ListAssets(p.ctx, request))
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		request := &assetpb.ListAssetsRequest{
			Parent:      p.config.Parent,
			AssetTypes:  assetTypes,
			ContentType: assetpb.ContentType_IAM_POLICY,
		}
		policyAssets = getAllAssets(p.log, p.inventory.ListAssets(p.ctx, request))
		wg.Done()
	}()
	wg.Wait()

	var assets []*assetpb.Asset
	assets = append(append(assets, resourceAssets...), policyAssets...)
	mergedAssets := mergeAssetContentType(assets)
	extendedAssets := extendAssets(p.ctx, p.crm, p.crmCache, mergedAssets)
	// Enrich network assets with dns policy
	p.enrichNetworkAssets(extendedAssets)

	return extendedAssets, nil
}

// ListMonitoringAssets returns a list of monitoring assets grouped by project id
func (p *Provider) ListMonitoringAssets(monitoringAssetTypes map[string][]string) ([]*MonitoringAsset, error) {
	logMetrics, err := p.ListAllAssetTypesByName(monitoringAssetTypes["LogMetric"])
	if err != nil {
		return nil, err
	}
	alertPolicies, err := p.ListAllAssetTypesByName(monitoringAssetTypes["AlertPolicy"])
	if err != nil {
		return nil, err
	}
	var assets []*ExtendedGcpAsset
	assets = append(append(assets, logMetrics...), alertPolicies...)
	monitoringAssets := getMonitoringAssetsByProject(assets, p.log)
	return monitoringAssets, nil
}

// ListLoggingAssets returns a list of logging assets grouped by project id, extended with folder and org level log sinks
func (p *Provider) ListLoggingAssets() ([]*LoggingAsset, error) {
	logSinks, err := p.ListAllAssetTypesByName([]string{LogSinkAssetType})
	if err != nil {
		return nil, err
	}

	loggingAssets := getLoggingAssetsByProject(logSinks, p.log)
	return loggingAssets, nil
}

// ListServiceUsageAssets returns a list of service usage assets grouped by project id
func (p *Provider) ListServiceUsageAssets() ([]*ServiceUsageAsset, error) {
	services, err := p.ListAllAssetTypesByName([]string{"serviceusage.googleapis.com/Service"})
	if err != nil {
		return nil, err
	}

	assets := getServiceUsageAssetsByProject(services, p.log)
	return assets, nil
}

func (p *Provider) Close() error {
	return p.inventory.Close()
}

// enrichNetworkAssets enriches the network assets with dns policy if exists
func (p *Provider) enrichNetworkAssets(assets []*ExtendedGcpAsset) {
	networkAssets := getAssetsByType(assets, ComputeNetworkAssetType)
	if len(networkAssets) == 0 {
		p.log.Infof("no %s assets were listed", ComputeNetworkAssetType)
		return
	}

	dnsPolicyAssets := getAllAssets(p.log, p.inventory.ListAssets(p.ctx, &assetpb.ListAssetsRequest{
		Parent:      p.config.Parent,
		AssetTypes:  []string{DnsPolicyAssetType},
		ContentType: assetpb.ContentType_RESOURCE,
	}))

	if len(dnsPolicyAssets) == 0 {
		p.log.Infof("no %s assets were listed, return original assets", DnsPolicyAssetType)
		return
	}

	dnsPolicies := decodeDnsPolicies(dnsPolicyAssets)

	p.log.Infof("attempting to enrich %d %s assets with dns policy", len(assets), ComputeNetworkAssetType)
	for _, networkAsset := range networkAssets {
		networkAssetFields := networkAsset.GetResource().GetData().GetFields()
		networkIdentifier := strings.TrimPrefix(networkAsset.GetName(), "//compute.googleapis.com")

		dnsPolicy := findDnsPolicyByNetwork(dnsPolicies, networkIdentifier)
		if dnsPolicy != nil {
			p.log.Infof("enrich a %s asset with dns policy, name: %s", ComputeNetworkAssetType, networkIdentifier)
			networkAssetFields["enabledDnsLogging"] = &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: dnsPolicy.enableLogging}}
		}
	}
}

// findDnsPolicyByNetwork finds DNS policy by network identifier
func findDnsPolicyByNetwork(dnsPolicies []*dnsPolicyFields, networkIdentifier string) *dnsPolicyFields {
	for _, dnsPolicy := range dnsPolicies {
		if lo.SomeBy(dnsPolicy.networks, func(networkUrl string) bool {
			return strings.HasSuffix(networkUrl, networkIdentifier)
		}) {
			return dnsPolicy
		}
	}
	return nil
}

// decodeDnsPolicies gets the required fields from the dns policies assets
func decodeDnsPolicies(dnsPolicyAssets []*assetpb.Asset) []*dnsPolicyFields {
	dnsPolicies := make([]*dnsPolicyFields, 0)
	for _, dnsPolicyAsset := range dnsPolicyAssets {
		fields := new(dnsPolicyFields)
		dnsPolicyData := dnsPolicyAsset.GetResource().GetData().GetFields()

		if attachedNetworks, exist := dnsPolicyData["networks"]; exist {
			networks := attachedNetworks.GetListValue().GetValues()
			for _, network := range networks {
				if networkUrl, found := network.GetStructValue().GetFields()["networkUrl"]; found {
					fields.networks = append(fields.networks, networkUrl.GetStringValue())
				}
			}
		}

		if enableLogging, exist := dnsPolicyData["enableLogging"]; exist {
			fields.enableLogging = enableLogging.GetBoolValue()
		}

		dnsPolicies = append(dnsPolicies, fields)
	}

	return dnsPolicies
}

// returns monitoring assets grouped by project id
// single project for project scoped accounts
// multiple projects for organization scoped accounts
func getMonitoringAssetsByProject(assets []*ExtendedGcpAsset, log *logp.Logger) []*MonitoringAsset {
	assetsByProject := lo.GroupBy(assets, func(asset *ExtendedGcpAsset) string { return asset.Ecs.ProjectId })
	var monitoringAssets []*MonitoringAsset
	for projectId, projectAssets := range assetsByProject {
		projectName, organizationId, organizationName, err := getProjectAssetsMetadata(projectAssets)
		if err != nil {
			log.Error(err)
			continue
		}
		monitoringAssets = append(monitoringAssets, &MonitoringAsset{
			LogMetrics: getAssetsByType(projectAssets, MonitoringLogMetricAssetType),
			Alerts:     getAssetsByType(projectAssets, MonitoringAlertPolicyAssetType),
			Ecs: &fetching.EcsGcp{
				Provider:         "gcp",
				ProjectId:        projectId,
				ProjectName:      projectName,
				OrganizationId:   organizationId,
				OrganizationName: organizationName,
			},
		})
	}
	return monitoringAssets
}

// returns logging assets grouped by project id
// single project for project scoped accounts
// multiple projects for organization scoped accounts
func getLoggingAssetsByProject(assets []*ExtendedGcpAsset, log *logp.Logger) []*LoggingAsset {
	assetsByProject := lo.GroupBy(assets, func(asset *ExtendedGcpAsset) string { return asset.Ecs.ProjectId })
	var loggingAssets []*LoggingAsset
	for projectId, projectAssets := range assetsByProject {
		if projectId == "" {
			continue
		}

		projectName, organizationId, organizationName, err := getProjectAssetsMetadata(projectAssets)
		if err != nil {
			log.Error(err)
			continue
		}

		// add folder and org level log sinks for each project
		projectAssets = append(projectAssets, assetsByProject[""]...)

		loggingAssets = append(loggingAssets, &LoggingAsset{
			LogSinks: projectAssets,
			Ecs: &fetching.EcsGcp{
				Provider:         "gcp",
				ProjectId:        projectId,
				ProjectName:      projectName,
				OrganizationId:   organizationId,
				OrganizationName: organizationName,
			},
		})
	}
	return loggingAssets
}

// returns service usage assets grouped by project id
// single project for project scoped accounts
// multiple projects for organization scoped accounts
func getServiceUsageAssetsByProject(assets []*ExtendedGcpAsset, log *logp.Logger) []*ServiceUsageAsset {
	assetsByProject := lo.GroupBy(assets, func(asset *ExtendedGcpAsset) string { return asset.Ecs.ProjectId })
	var serviceUsageAssets []*ServiceUsageAsset
	for projectId, projectAssets := range assetsByProject {
		projectName, organizationId, organizationName, err := getProjectAssetsMetadata(projectAssets)
		if err != nil {
			log.Error(err)
			continue
		}
		serviceUsageAssets = append(serviceUsageAssets, &ServiceUsageAsset{
			Services: projectAssets,
			Ecs: &fetching.EcsGcp{
				Provider:         "gcp",
				ProjectId:        projectId,
				ProjectName:      projectName,
				OrganizationId:   organizationId,
				OrganizationName: organizationName,
			},
		})
	}
	return serviceUsageAssets
}

func getAllAssets(log *logp.Logger, it Iterator) []*assetpb.Asset {
	results := make([]*assetpb.Asset, 0)
	for {
		response, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Errorf("Error fetching GCP Asset: %s", err)
			return nil
		}

		log.Debugf("Fetched GCP Asset: %+v", response.Name)
		results = append(results, response)
	}
	return results
}

func mergeAssetContentType(assets []*assetpb.Asset) []*assetpb.Asset {
	var resultsMap = make(map[string]*assetpb.Asset)
	for _, asset := range assets {
		assetKey := asset.Name
		if _, ok := resultsMap[assetKey]; !ok {
			resultsMap[assetKey] = asset
			continue
		}
		item := resultsMap[assetKey]
		if asset.Resource != nil {
			item.Resource = asset.Resource
		}
		if asset.IamPolicy != nil {
			item.IamPolicy = asset.IamPolicy
		}
	}
	var results []*assetpb.Asset
	for _, asset := range resultsMap {
		results = append(results, asset)
	}
	return results
}

// extends the assets with the project and organization display name
func extendAssets(ctx context.Context, crm *ResourceManagerWrapper, cache map[string]*fetching.EcsGcp, assets []*assetpb.Asset) []*ExtendedGcpAsset {
	var extendedAssets []*ExtendedGcpAsset
	for _, asset := range assets {
		keys := getAssetIds(asset)
		cacheKey := fmt.Sprintf("%s/%s", keys.parentProject, keys.parentOrg)
		if ecsGcpCloudCached, ok := cache[cacheKey]; ok {
			extendedAssets = append(extendedAssets, &ExtendedGcpAsset{
				Asset: asset,
				Ecs:   ecsGcpCloudCached,
			})
			continue
		}
		cache[cacheKey] = getEcsGcpCloudData(ctx, crm, keys)
		extendedAssets = append(extendedAssets, &ExtendedGcpAsset{
			Asset: asset,
			Ecs:   cache[cacheKey],
		})
	}
	return extendedAssets
}

func getAssetIds(asset *assetpb.Asset) GcpAssetIDs {
	orgId := getOrganizationId(asset.Ancestors)
	projectId := getProjectId(asset.Ancestors)
	parentProject := fmt.Sprintf("projects/%s", projectId)
	parentOrg := fmt.Sprintf("organizations/%s", orgId)
	return GcpAssetIDs{
		orgId:         orgId,
		projectId:     projectId,
		parentProject: parentProject,
		parentOrg:     parentOrg,
	}
}

func getEcsGcpCloudData(ctx context.Context, crm *ResourceManagerWrapper, keys GcpAssetIDs) *fetching.EcsGcp {
	var orgName string
	var projectName string
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		orgName = crm.getOrganizationDisplayName(ctx, keys.parentOrg)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		projectName = crm.getProjectDisplayName(ctx, keys.parentProject)
		wg.Done()
	}()
	wg.Wait()
	return &fetching.EcsGcp{
		ProjectId:        keys.projectId,
		ProjectName:      projectName,
		OrganizationId:   keys.orgId,
		OrganizationName: orgName,
	}
}

func getOrganizationId(ancestors []string) string {
	last := ancestors[len(ancestors)-1]
	parts := strings.Split(last, "/") // organizations/1234567890

	if parts[0] == "organizations" {
		return parts[1]
	}

	return ""
}

func getProjectId(ancestors []string) string {
	parts := strings.Split(ancestors[0], "/") // projects/1234567890

	if parts[0] == "projects" {
		return parts[1]
	}

	return ""
}

func getAssetsByType(projectAssets []*ExtendedGcpAsset, assetType string) []*ExtendedGcpAsset {
	return lo.Filter(projectAssets, func(asset *ExtendedGcpAsset, _ int) bool {
		return asset.AssetType == assetType
	})
}

func getProjectAssetsMetadata(projectAssets []*ExtendedGcpAsset) (string, string, string, error) {
	if len(projectAssets) == 0 {
		return "", "", "", errors.New("failed to get project/organization name")
	}
	// We grouped the assets by project id, so we can get the project metadata from the first asset
	asset := projectAssets[0]
	return asset.Ecs.ProjectName,
		asset.Ecs.OrganizationId,
		asset.Ecs.OrganizationName,
		nil
}
