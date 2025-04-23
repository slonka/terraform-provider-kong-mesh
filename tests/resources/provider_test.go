package tests

import (
    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/hashicorp/terraform-plugin-go/tfprotov6"
    "github.com/kong/terraform-provider-kong-mesh/internal/provider"
)

var (
    providerFactory = map[string]func() (tfprotov6.ProviderServer, error){
        "kong-mesh": providerserver.NewProtocol6WithError(provider.New("")()),
    }
)
