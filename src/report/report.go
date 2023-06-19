package report

import (
	cfg "github.com/accuknox/auto-policy-discovery/src/config"
	"github.com/accuknox/auto-policy-discovery/src/libs"
	opb "github.com/accuknox/auto-policy-discovery/src/protobuf/v1/observability"
	rpb "github.com/accuknox/auto-policy-discovery/src/protobuf/v1/report"
	"github.com/accuknox/auto-policy-discovery/src/types"
	"strconv"
)

type Config struct {
	CfgDB types.ConfigDB
}

func InitializeConfig() {
	Rcfg = &Config{CfgDB: cfg.GetCfgDB()}
}

var Rcfg *Config

type Options struct {
	options *types.ReportOptions
}

func (o *Options) GetReport() (*rpb.ReportResponse, error) {

	res, err := getSystemReport(o)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func getSystemReport(o *Options) (*rpb.ReportResponse, error) {

	res := &rpb.ReportResponse{}
	clusters := &rpb.ClusterData{}
	namespace := &rpb.NamespaceData{}

	reportData, err := getKubearmorReportData(Rcfg.CfgDB, o.options)
	if err != nil {
		return nil, err
	}

	for ck, cv := range reportData.Cluster {
		clusters.ClusterName = ck
		for nk, nv := range cv.Namespace {
			namespace.NamespaceName = nk
			namespace.Resources = map[string]*rpb.ResourceData{}
			for _, rv := range nv.ResourceSummaryData {
				namespace.Resources[rv.ResourceType] = &rpb.ResourceData{}
				namespace.Resources[rv.ResourceType].ResourceName = rv.ResourceName
				namespace.Resources[rv.ResourceType].MData = &rpb.MetaData{
					Label:         rv.MetaData.Label,
					ContainerName: rv.MetaData.ContainerName,
				}

				for _, sd := range rv.SummaryData.ProcessData {
					namespace.Resources[rv.ResourceType].SumData.ProcessData = append(namespace.Resources[rv.ResourceType].SumData.ProcessData,
						&opb.SysProcFileSummaryData{
							Source:      sd.Source,
							Destination: sd.Destination,
							Status:      sd.Status,
						})
				}

				for _, sd := range rv.SummaryData.FileData {
					namespace.Resources[rv.ResourceType].SumData.FileData = append(namespace.Resources[rv.ResourceType].SumData.FileData,
						&opb.SysProcFileSummaryData{
							Source:      sd.Source,
							Destination: sd.Destination,
							Status:      sd.Status,
						})
				}

				for _, sd := range rv.SummaryData.NetworkData {
					if sd.NetType == "ingress" {
						namespace.Resources[rv.ResourceType].SumData.IngressConnection = append(namespace.Resources[rv.ResourceType].SumData.IngressConnection, &opb.SysNwSummaryData{
							Protocol:  sd.Protocol,
							Command:   sd.Command,
							IP:        sd.PodSvcIP,
							Port:      sd.ServerPort,
							Labels:    sd.Labels,
							Namespace: sd.Namespace,
						})
					} else if sd.NetType == "egress" {
						namespace.Resources[rv.ResourceType].SumData.EgressConnection = append(namespace.Resources[rv.ResourceType].SumData.EgressConnection, &opb.SysNwSummaryData{
							Protocol:  sd.Protocol,
							Command:   sd.Command,
							IP:        sd.PodSvcIP,
							Port:      sd.ServerPort,
							Labels:    sd.Labels,
							Namespace: sd.Namespace,
						})
					} else if sd.NetType == "bind" {
						namespace.Resources[rv.ResourceType].SumData.BindConnection = append(namespace.Resources[rv.ResourceType].SumData.BindConnection, &opb.SysNwSummaryData{
							Protocol:    sd.Protocol,
							Command:     sd.Command,
							IP:          sd.PodSvcIP,
							BindPort:    sd.BindPort,
							BindAddress: sd.BindAddress,
							Labels:      sd.Labels,
							Namespace:   sd.Namespace,
						})
					}
				}
			}
		}
		res.Clusters = append(res.Clusters, clusters)
	}
	return res, nil
}

func getKubearmorReportData(CfgDB types.ConfigDB, reportOptions *types.ReportOptions) (*ReportData, error) {
	var err error
	var processData, fileData []types.SysObsProcFileData
	var nwData []types.SysObsNwData
	var reportSummaryData ReportData
	var sysSummary []types.SystemSummary

	sysSummary, err = libs.GetSystemSummary(CfgDB, nil, reportOptions)

	if err != nil {
		return nil, err
	}

	for _, ss := range sysSummary {

		_, ok := reportSummaryData.Cluster[ss.ClusterName]
		if !ok {
			reportSummaryData.Cluster[ss.ClusterName] = Clusters{
				NamespaceName: []string{ss.NamespaceName},
				Namespace:     map[string]ResourceTypeData{},
			}
			reportSummaryData.Cluster[ss.ClusterName].Namespace[ss.NamespaceName] = ResourceTypeData{
				ResourceType:        "Deployment",
				ResourceSummaryData: map[string]ResourceData{},
			}
			reportSummaryData.Cluster[ss.ClusterName].Namespace[ss.NamespaceName].ResourceSummaryData["Deployment"] = ResourceData{
				ResourceType: "Deployment",
				ResourceName: ss.Deployment,
				MetaData: &types.MetaData{
					Label:         ss.Labels,
					ContainerName: ss.ContainerName,
				},
				SummaryData: &SummaryData{
					ProcessData: processData,
					FileData:    fileData,
					NetworkData: nwData,
				},
			}
		}
		//t := time.Unix(ss.UpdatedTime, 0)

		if ss.Operation == "Process" {
			//ExtractProcessData
			processData = append(processData, types.SysObsProcFileData{
				Source:      ss.Source,
				Destination: ss.Destination,
				Status:      ss.Action,
				//Count:       uint32(ss.Count),
				//: t.Format(time.UnixDate),
			})
		} else if ss.Operation == "File" {
			//ExtractFileData
			fileData = append(fileData, types.SysObsProcFileData{
				Source:      ss.Source,
				Destination: ss.Destination,
				Status:      ss.Action,
				//:       uint32(ss.Count),
				//UpdatedTime: t.Format(time.UnixDate),
			})
		} else if ss.Operation == "Network" {
			//ExtractNwData
			nwData = append(nwData, types.SysObsNwData{
				NetType:     ss.NwType,
				Protocol:    ss.Protocol,
				Command:     ss.Source,
				PodSvcIP:    ss.IP,
				ServerPort:  strconv.Itoa(int(ss.Port)),
				BindPort:    ss.BindPort,
				BindAddress: ss.BindAddress,
				Namespace:   ss.DestNamespace,
				Labels:      ss.DestLabels,
				//Count:       uint32(ss.Count),
				//UpdatedTime: t.Format(time.UnixDate),
			})
		}

		reportSummaryData.Cluster[ss.ClusterName].Namespace[ss.NamespaceName].ResourceSummaryData["Deployment"].SummaryData.ProcessData = processData
		reportSummaryData.Cluster[ss.ClusterName].Namespace[ss.NamespaceName].ResourceSummaryData["Deployment"].SummaryData.FileData = fileData
		reportSummaryData.Cluster[ss.ClusterName].Namespace[ss.NamespaceName].ResourceSummaryData["Deployment"].SummaryData.NetworkData = nwData

	}

	return &reportSummaryData, nil

}
