"""A template file for deploying a compute engine instance."""

COMPUTE_URL_BASE = "https://www.googleapis.com/compute/v1/"


def generate_config(context):
    """Generates configuration."""
    project = context.env["project"]
    deployment_name = context.env["deployment"]
    zone = context.properties["zone"]
    enrollment_token = context.properties["enrollmentToken"]
    fleet_url = context.properties["fleetUrl"]
    agent_version = context.properties["elasticAgentVersion"]
    artifact_server = context.properties["elasticArtifactServer"]
    scope = context.properties["scope"]
    parent_id = context.properties["parentId"]

    roles = ["roles/cloudasset.viewer", "roles/browser"]
    network_name = f"{deployment_name}-network"
    sa_name = f"{deployment_name}-sa"

    ssh_fw_rule = {
        "name": "elastic-agent-firewall-rule",
        "type": "compute.v1.firewall",
        "properties": {
            "network": f"$(ref.{network_name}.selfLink)",
            "sourceRanges": ["0.0.0.0/0"],
            "allowed": [
                {
                    "IPProtocol": "TCP",
                    "ports": [22],
                },
            ],
        },
    }

    instance = {
        "name": deployment_name,
        "type": "compute.v1.instance",
        "properties": {
            "zone": zone,
            "machineType": "".join(
                [
                    COMPUTE_URL_BASE,
                    "projects/",
                    project,
                    "/zones/",
                    zone,
                    "/",
                    "machineTypes/n2-standard-4",
                ],
            ),
            "serviceAccounts": [
                {
                    "email": f"$(ref.{sa_name}.email)",
                    "scopes": ["https://www.googleapis.com/auth/cloud-platform"],
                },
            ],
            "disks": [
                {
                    "deviceName": "boot",
                    "type": "PERSISTENT",
                    "boot": True,
                    "sizeGb": 32,
                    "autoDelete": True,
                    "initializeParams": {
                        "sourceImage": "".join(
                            [
                                COMPUTE_URL_BASE,
                                "projects/",
                                "ubuntu-os-cloud/global",
                                "/images/family/ubuntu-minimal-2204-lts",
                            ],
                        ),
                    },
                },
            ],
            "metadata": {
                "dependsOn": [sa_name],
                "items": [
                    {
                        "key": "startup-script",
                        "value": "".join(
                            [
                                "#!/bin/bash\n",
                                "set -x\n",
                                f"ElasticAgentArtifact=elastic-agent-{agent_version}-linux-x86_64\n",
                                f"curl -L -O {artifact_server}/$ElasticAgentArtifact.tar.gz\n",
                                "tar xzvf $ElasticAgentArtifact.tar.gz\n",
                                "cd $ElasticAgentArtifact\n",
                                f"sudo ./elastic-agent install "
                                f"--non-interactive --url={fleet_url} --enrollment-token={enrollment_token}",
                            ],
                        ),
                    },
                ],
            },
            "networkInterfaces": [
                {
                    "network": f"$(ref.{network_name}.selfLink)",
                    "accessConfigs": [
                        {
                            "name": "External NAT",
                            "type": "ONE_TO_ONE_NAT",
                        },
                    ],
                },
            ],
            "labels": {
                "name": "elastic-agent",
            },
        },
    }

    service_account = {
        "name": sa_name,
        "type": "iam.v1.serviceAccount",
        "properties": {
            "accountId": sa_name,
            "displayName": "Elastic agent service account for CSPM",
            "projectId": context.env["project"],
        },
    }

    network = {
        "name": network_name,
        "type": "compute.v1.network",
        "properties": {
            "routingConfig": {
                "routingMode": "REGIONAL",
            },
            "autoCreateSubnetworks": True,
        },
    }

    bindings = []
    for role in roles:
        bindings.append(
            {
                "name": f"{deployment_name}-iam-binding-{role}",
                "type": f"gcp-types/cloudresourcemanager-v1:virtual.{scope}.iamMemberBinding",
                "properties": {
                    "resource": parent_id,
                    "role": role,
                    "member": f"serviceAccount:$(ref.{sa_name}.email)",
                },
                "metadata": {
                    "dependsOn": [sa_name],
                },
            },
        )

    resources = [instance, service_account, network]
    resources.extend(bindings)

    if context.properties["allowSSH"]:
        resources.append(ssh_fw_rule)

    return {"resources": resources}
