package tests

import (
    "fmt"
    "github.com/hashicorp/terraform-plugin-testing/plancheck"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
    "io"
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMesh(t *testing.T) {
    ctx := t.Context()
    req := testcontainers.ContainerRequest{
        Image:        "kong/kuma-cp:2.10.1",
        ExposedPorts: []string{"5681/tcp"},
        WaitingFor:   wait.ForListeningPort("5681/tcp"),
        Cmd:          []string{"run"},
    }
    cpContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    require.NoError(t, err)
    defer testcontainers.CleanupContainer(t, cpContainer)
    port, err := cpContainer.MappedPort(ctx, "5681/tcp")
    require.NoError(t, err)

    t.Run("creates a mesh", func(t *testing.T) {
        meshName := "terraform-provider-kong-mesh"
        meshResource := "default"
        providerName := "kong-mesh"

        resource.Test(t, resource.TestCase{
            ProtoV6ProviderFactories: providerFactory,
            Steps: []resource.TestStep{
                {
                    Config: providerConfig(port.Port()) +
                        mesh(providerName, meshResource, meshName),
                    ConfigPlanChecks: resource.ConfigPlanChecks{
                        PreApply: []plancheck.PlanCheck{
                            plancheck.ExpectResourceAction(resourceAddress(providerName, "mesh", meshResource), plancheck.ResourceActionCreate),
                        },
                    },
                },
                {
                    // Re-apply the same config and ensure no changes occur
                    Config: providerConfig(port.Port()) +
                        mesh(providerName, meshResource, meshName),
                    ConfigPlanChecks: resource.ConfigPlanChecks{
                        PreApply: []plancheck.PlanCheck{
                            plancheck.ExpectEmptyPlan(),
                        },
                    },
                },
            },
        })
    })

    logs, err := cpContainer.Logs(ctx)
    require.NoError(t, err)
    defer logs.Close()
    logContent, err := io.ReadAll(logs)
    require.NoError(t, err)
    t.Logf("Container logs: %s", logContent)
}

func mesh(providerName, resourceName, meshName string) string {
    return fmt.Sprintf(`resource "%s_mesh" "%s" {
  type  = "Mesh"
  name  = "%s"
}
`, providerName, resourceName, meshName)
}

func resourceAddress(providerName, resourceType, resourceName string) string {
    return fmt.Sprintf("%s_%s.%s", providerName, resourceType, resourceName)
}
