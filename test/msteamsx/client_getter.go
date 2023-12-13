package msteamsx

import (
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

type GraphAPIClientGetter struct {
	appID       string
	appPassword string

	clients sync.Map
}

func NewGraphAPIClientGetter(appID string, appPassword string) *GraphAPIClientGetter {
	return &GraphAPIClientGetter{
		appID:       appID,
		appPassword: appPassword,
	}
}

func (g *GraphAPIClientGetter) GetForTenant(tenantID string) (*msgraphsdk.GraphServiceClient, error) {
	cli, ok := g.clients.Load(tenantID)
	if !ok {
		cli, err := g.newClientForTenant(tenantID)
		if err != nil {
			return nil, errors.Wrap(err, "while creating Graph API client")
		}

		g.clients.Store(tenantID, cli)
		return cli, nil
	}

	typedCli, ok := cli.(*msgraphsdk.GraphServiceClient)
	if !ok {
		return nil, fmt.Errorf("invalid Graph API client type %T", typedCli)
	}

	return typedCli, nil
}

func (g *GraphAPIClientGetter) newClientForTenant(tenantID string) (*msgraphsdk.GraphServiceClient, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, g.appID, g.appPassword, nil)
	if err != nil {
		return nil, errors.Wrap(err, "while creating Azure credentials")
	}
	graphClient, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, nil)
	if err != nil {
		return nil, errors.Wrap(err, "while creating Graph API client")
	}

	return graphClient, nil
}
