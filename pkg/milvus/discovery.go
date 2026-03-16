package milvus

import (
	"errors"
	"fmt"

	"github.com/criteo/blackbox-prober/pkg/common"
	"github.com/criteo/blackbox-prober/pkg/discovery"
	"github.com/criteo/blackbox-prober/pkg/topology"
	"github.com/criteo/blackbox-prober/pkg/utils"
	mv "github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func (conf *MilvusProbeConfig) buildAddress(tlsEnabled bool, addressUrl string) string {
	proto := "http"
	if tlsEnabled {
		proto = "https"
	}

	return fmt.Sprintf("%s://%s", proto, addressUrl)
}

func (conf *MilvusProbeConfig) generateClusterEndpointsFromEntry(logger log.Logger, entry discovery.ServiceEntry) ([]*MilvusEndpoint, error) {
	authEnabled := conf.MilvusEndpointConfig.AuthEnabled
	var (
		username string
		password string
		ok       bool
	)

	if authEnabled {
		var err error
		username, password, err = common.LoadBasicAuthCredentials(authEnabled, conf.MilvusEndpointConfig.UsernameEnv, conf.MilvusEndpointConfig.PasswordEnv)
		if err != nil {
			return nil, err
		}
	}
	tlsEnabled := utils.Contains(entry.Tags, conf.MilvusEndpointConfig.TLSTag)
	addressUrl, ok := entry.Meta[conf.MilvusEndpointConfig.AddressMetaKey]
	if !ok {
		msg := fmt.Sprintf("%s not found in consul meta key for service %s", conf.MilvusEndpointConfig.AddressMetaKey, entry.Service)
		level.Warn(logger).Log("msg", msg)
		return nil, errors.New(msg)
	}

	clusterName, ok := entry.Meta[conf.DiscoveryConfig.MetaClusterKey]
	if !ok {
		msg := fmt.Sprintf("ClusterName meta key not found. Ignoring service %s.", entry.Service)
		level.Error(logger).Log("msg", msg)
		return nil, errors.New(msg)
	}
	address := conf.buildAddress(tlsEnabled, addressUrl)

	endpoint := &MilvusEndpoint{Name: clusterName,
		ClusterName:  clusterName,
		ClusterLevel: true,
		ClientConfig: mv.ClientConfig{
			// auth
			Username: username,
			Password: password,
			// tls
			Address: address,
			// conf
			RetryRateLimit: &mv.RetryRateLimitOption{
				MaxRetry:   conf.MilvusEndpointConfig.MaxRetry,
				MaxBackoff: conf.MilvusEndpointConfig.MaxBackoff,
			},
		},
		Config: conf.MilvusEndpointConfig,
		Logger: log.With(logger, "endpoint_name", entry.Address),
	}

	return []*MilvusEndpoint{endpoint}, nil
}

func (conf *MilvusProbeConfig) BuildTopology(logger log.Logger, entries []discovery.ServiceEntry) (topology.ClusterMap, error) {
	clusterMap := topology.NewClusterMap()
	clusterEntries := conf.DiscoveryConfig.GroupNodesByCluster(logger, entries)
	for _, clusterGroup := range clusterEntries {
		endpoints, err := conf.generateClusterEndpointsFromEntry(logger, clusterGroup[0])
		if err != nil {
			return clusterMap, err
		}

		for _, endpoint := range endpoints {
			level.Debug(logger).Log("msg", "Adding cluster", "cluster", endpoint.Name, "address", endpoint.ClientConfig.Address)

			cluster := topology.NewCluster(endpoint)
			clusterMap.AppendCluster(cluster)
		}

	}
	return clusterMap, nil
}
