package tests

import (
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/kong/terraform-provider-kong-mesh/tests/tfbuilder"
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
		WaitingFor: wait.ForAll(
			wait.ForLog("default AccessRoleBinding created"),
			wait.ForLog("default AccessRole created"),
			wait.ForLog("saving generated Admin User Token"),
			wait.ForListeningPort("5681/tcp"),
		),
		Cmd: []string{"run"},
	}
	cpContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer testcontainers.CleanupContainer(t, cpContainer)
	port, err := cpContainer.MappedPort(ctx, "5681/tcp")
	require.NoError(t, err)

	t.Skip("creates a mesh without skip_creating_initial_policies", func(t *testing.T) {
		builder := tfbuilder.NewBuilder(tfbuilder.KongMesh, "http", "localhost", port.Int())
		mesh := tfbuilder.NewMeshBuilder("m1", "m1")
		builder.AddMesh(mesh)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: providerFactory,
			Steps: []resource.TestStep{
				{
					Config: builder.Build(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(builder.ResourceAddress("mesh", mesh.ResourceName), plancheck.ResourceActionCreate),
						},
					},
				},
				{
					// Re-apply the same config and ensure no changes occur
					Config: builder.Build(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
			},
		})
	})

	t.Run("creates a mesh with a policy", func(t *testing.T) {
		builder := tfbuilder.NewBuilder(tfbuilder.KongMesh, "http", "localhost", port.Int())
		mesh := tfbuilder.NewMeshBuilder("default", "terraform-provider-kong-mesh").
			WithSpec(`skip_creating_initial_policies = [ "*" ]`)
		mtp := tfbuilder.NewPolicyBuilder("mesh_traffic_permission", "allow_all", "allow-all", "MeshTrafficPermission").
			WithMeshRef(builder.ResourceAddress("mesh", mesh.ResourceName) + ".name").
			WithDependsOn(builder.ResourceAddress("mesh", mesh.ResourceName)).
			WithLabels(map[string]string{
				"kuma.io/mesh":   mesh.MeshName,
				"kuma.io/env":    "universal",
				"kuma.io/origin": "zone",
				"kuma.io/zone":   "default",
			}).
			WithSpecHCL(tfbuilder.AllowAllTrafficPermissionSpec)
		builder.AddMesh(mesh)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: providerFactory,
			Steps: []resource.TestStep{
				{
					Config: builder.Build(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(builder.ResourceAddress("mesh", mesh.ResourceName), plancheck.ResourceActionCreate),
						},
					},
				},
				{
					// Re-apply the same config and ensure no changes occur
					Config: builder.Build(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
				{
					Config: builder.AddPolicy(mtp).Build(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(builder.ResourceAddress(mtp.ResourceType, mtp.ResourceName), plancheck.ResourceActionCreate),
						},
					},
				},
				{
					// Re-apply the same config and ensure no changes occur
					Config: builder.Build(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
			},
		})
	})

	if t.Failed() {
		logs, err := cpContainer.Logs(ctx)
		require.NoError(t, err)
		defer logs.Close()
		logContent, err := io.ReadAll(logs)
		require.NoError(t, err)
		t.Logf("Container logs: %s", logContent)
	}
}
