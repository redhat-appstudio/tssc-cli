# Host Azure Pipelines build agent on openshift cluster

## Prerequisites

1. OpenShift 4 cluster

2. Azure DevOps organization

## Install Azure build agent on OpenShift

1. Follow the documentation to [Set up a Personal Access Token](https://learn.microsoft.com/en-us/azure/devops/pipelines/agents/linux-agent?view=azure-devops#authenticate-with-a-personal-access-token-pat) in Azure. Save the token for later use in installation.
2. Create a new Agent Pool or configure an existing Agent Pool by navigating to Organization Settings -> Agent Pools.
   - Click on Add pool
   - Pool type: Self-hosted
   - Name: <any-name>
   - Pipeline permissions: Grant access permission to all pipelines
3. Login to OpenShift cluster
4. `cd` to the directory that contains this README
5. Navigate to the [Azure Pipeline Agent](https://github.com/Microsoft/azure-pipelines-agent/releases) and check for Linux x86 Agent download url latest version. Modify [AZP_AGENT_PACKAGE_LATEST_URL](./azure-buildconfig.yaml#L44) accordingly.
6. Replace the dummy required credential for Azure DevOps in [./azure-deploy.sh](./azure-deploy.sh#L21-L23) for authentication.
7. Run the [./azure-deploy.sh](./azure-deploy.sh) script.
8. Verify the azure build agent is running
   - In Azure DevOps portal, navigate to Organization Settings -> Agent Pools -> Default (or your own pool) -> Agents
   - You should now see the build agent with Online status.
   - To deploy additional agents, you can scale up the pod replicas.
