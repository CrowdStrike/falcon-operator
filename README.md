![CrowdStrike Falcon](https://raw.githubusercontent.com/CrowdStrike/falconpy/main/docs/asset/cs-logo.png) [![Twitter URL](https://img.shields.io/twitter/url?label=Follow%20%40CrowdStrike&style=social&url=https%3A%2F%2Ftwitter.com%2FCrowdStrike)](https://twitter.com/CrowdStrike)<br/>

# Falcon Operator
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/falcon-operator)](https://artifacthub.io/packages/search?repo=falcon-operator)
[![CI Golang Build](https://github.com/CrowdStrike/falcon-operator/actions/workflows/go.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/crowdstrike/falcon-operator)](https://goreportcard.com/report/github.com/crowdstrike/falcon-operator)
[![CodeQL](https://github.com/CrowdStrike/falcon-operator/actions/workflows/codeql.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/crowdstrike/falcon-operator.svg)](https://pkg.go.dev/github.com/crowdstrike/falcon-operator)
[![CI Container Build](https://github.com/CrowdStrike/falcon-operator/actions/workflows/container_build.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/container_build.yml)
[![Docker Repository on Quay](https://quay.io/repository/crowdstrike/falcon-operator/status "Docker Repository on Quay")](https://quay.io/repository/crowdstrike/falcon-operator)
[![Docker Repository on Quay](https://quay.io/repository/crowdstrike/falcon-operator-bundle/status "Docker Repository on Quay")](https://quay.io/repository/crowdstrike/falcon-operator-bundle)

The CrowdStrike Falcon Operator installs CrowdStrike Falcon custom resources on a Kubernetes cluster.

The CrowdStrike Falcon Operator is an open source project and not a CrowdStrike product. As such, it carries no formal support, expressed, or implied.

## About the CrowdStrike Falcon Operator
The CrowdStrike Falcon Operator deploys CrowdStrike Falcon to the cluster. The operator exposes custom resources that allow you to protect your Kubernetes clusters when deployed.

## About Custom Resources

| Custom Resource                                       | Description                                                      |
| :--------                                             | :------------                                                    |
| [FalconAdmission](https://github.com/CrowdStrike/falcon-operator/tree/main/docs/resources/admission/README.md) | Manages installation of Falcon Admission Controller on the cluster |
| [FalconImageAnalyzer](https://github.com/CrowdStrike/falcon-operator/tree/main/docs/resources/imageanalyzer/README.md) | Manages installation of Falcon Image Assessment at Runtime on the cluster |
| [FalconContainer](https://github.com/CrowdStrike/falcon-operator/tree/main/docs/resources/container/README.md) | Manages installation of Falcon Container Sensor on the cluster   |
| [FalconNodeSensor](https://github.com/CrowdStrike/falcon-operator/tree/main/docs/resources/node/README.md)     | Manages installation of Falcon Linux Sensor on the cluster nodes |


## Installation and Deployment

For installation and deployment of the CrowdStrike Falcon Operator and its Custom Resources, please read the [Installation and Deployment Guide](docs/install_guide.md) and choose the deployment method that is right for your target environment.

## Getting Help
If you encounter any issues while using the Falcon Operator, you can create an issue on our [Github repo](https://github.com/CrowdStrike/falcon-operator) for bugs, enhancements, or other requests.

## Contributing
You can contribute by:

* Raising any issues you find using Falcon Operator
* Fixing issues by opening [Pull Requests](https://github.com/CrowdStrike/falcon-operator/pulls)
* Improving documentation
* Talking about the CrowdStrike Falcon Operator

All bugs, tasks or enhancements are tracked as [GitHub issues](https://github.com/CrowdStrike/falcon-operator/issues).

## Additional Resources
 - CrowdStrike Container Security: [Product Page](https://www.crowdstrike.com/products/cloud-security/falcon-cloud-workload-protection/container-security/)
 - So You Think Your Containers Are Secure? Four Steps to Ensure a Secure Container Deployment: [Blog Post](https://www.crowdstrike.com/blog/four-steps-to-ensure-a-secure-containter-deployment/)
 - Container Security With CrowdStrike: [Blog Post](https://www.crowdstrike.com/blog/tech-center/container-security/)
 - To learn more about Falcon Container Sensor for Linux: [Deployment Guide](https://falcon.crowdstrike.com/support/documentation/146/falcon-container-sensor-for-linux), [Release Notes](https://falcon.crowdstrike.com/support/news/release-notes-falcon-container-sensor-for-linux)
 - To learn more about Falcon Sensor for Linux: [Deployment Guide](https://falcon.crowdstrike.com/documentation/20/falcon-sensor-for-linux#kubernetes-support)
 - [Developer Documentation](docs/developer_guide.md)
