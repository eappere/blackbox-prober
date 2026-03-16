package milvus

import (
	"context"
	"fmt"
	"time"

	"github.com/criteo/blackbox-prober/pkg/common"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	mv "github.com/milvus-io/milvus/client/v2/milvusclient"
)

type MilvusEndpoint struct {
	Name         string
	ClusterLevel bool
	ClusterName  string
	Client       *mv.Client
	ClientConfig mv.ClientConfig
	Config       MilvusEndpointConfig
	Logger       log.Logger
	reauthState  common.ReauthState
}

func (e *MilvusEndpoint) GetHash() string {
	return fmt.Sprintf("%s/%s/db:%s", e.ClusterName, e.Name, e.Config.MonitoringDatabase)
}

func (e *MilvusEndpoint) GetName() string {
	return e.Name
}

func (e *MilvusEndpoint) IsCluster() bool {
	return e.ClusterLevel
}

func (e *MilvusEndpoint) Connect() error {
	username, password, err := common.LoadBasicAuthCredentials(e.Config.AuthEnabled, e.Config.UsernameEnv, e.Config.PasswordEnv)
	if err != nil {
		return err
	}
	e.ClientConfig.Username = username
	e.ClientConfig.Password = password

	// TODO: maybe make timeout configurable? For now hardcoding to 15s should be quite okay
	context, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*15))
	defer cancel()
	client, err := mv.New(context, &e.ClientConfig)
	if err != nil {
		return err
	}
	e.Client = client
	e.reauthState.MarkConnected(time.Now())
	return nil
}

func (e *MilvusEndpoint) Refresh() error {
	if e.Client == nil {
		return e.Connect()
	}

	reauthed, err := common.ReauthIfNeeded(time.Now(), e.Config.ReauthInterval, &e.reauthState, e.Close, e.Connect)
	if err != nil {
		return err
	}
	if reauthed {
		level.Info(e.Logger).Log("msg", "Reauthenticated Milvus endpoint connection", "interval", e.Config.ReauthInterval)
	}
	return nil
}

func (e *MilvusEndpoint) Close() error {
	if e != nil && e.Client != nil {
		e.Client.Close(context.Background()) // no timeout on close
		e.Client = nil
	}
	return nil
}
